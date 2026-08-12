package laravel

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type QueueMetrics struct {
	Driver          *string  `json:"driver"`
	Size            *int     `json:"size"`
	Pending         *int     `json:"pending"`
	Scheduled       *int     `json:"scheduled"`
	Reserved        *int     `json:"reserved"`
	OldestPending   *int     `json:"oldest_pending"`
	Failed          *int     `json:"failed"`
	OldestFailed    *int     `json:"oldest_failed"`
	NewestFailed    *int     `json:"newest_failed"`
	Failed1Min      *int     `json:"failed_1m"`
	Failed5Min      *int     `json:"failed_5m"`
	Failed10Min     *int     `json:"failed_10m"`
	FailedRate1Min  *float32 `json:"failed_rate_1m"`
	FailedRate5Min  *float32 `json:"failed_rate_5m"`
	FailedRate10Min *float32 `json:"failed_rate_10m"`
	ParseError      any      `json:"error"`
}

type QueueSizes map[string]map[string]QueueMetrics

func GetQueueSizes(ctx context.Context, appPath string, phpBinary string, queueMap map[string][]string) (*QueueSizes, error) {
	if len(queueMap) == 0 {
		return &QueueSizes{}, nil
	}

	script, err := buildQueueScript(queueMap)
	if err != nil {
		return nil, err
	}

	return runQueueScript(ctx, appPath, phpBinary, script)
}

// buildQueueScript renders the tinker script that reports queue sizes.
//
// Connection and queue names come from user configuration, so they are handed to
// PHP as base64-encoded JSON rather than interpolated into the script. A name
// containing a quote used to terminate the surrounding PHP string, which broke
// the script at best and executed arbitrary PHP at worst.
func buildQueueScript(queueMap map[string][]string) (string, error) {
	encoded, err := json.Marshal(queueMap)
	if err != nil {
		return "", fmt.Errorf("failed to encode queue config: %w", err)
	}

	script := fmt.Sprintf(`use Illuminate\Queue\QueueManager;
use Illuminate\Queue\Failed\FailedJobProviderInterface;
use Carbon\Carbon;

$manager = app(QueueManager::class);
$failedJobsProvider = app(FailedJobProviderInterface::class);
$now = now();
$sizes = [];
$queues = json_decode(base64_decode('%s'), true);`, base64.StdEncoding.EncodeToString(encoded))

	script += `
foreach ($queues as $conn => $qs) {
foreach ($qs as $q) {
	try {
		$sizes[$conn][$q] = ['size' => null, 'pending' => null, 'delayed' => null, 'oldest_pending' => null, 'failed' => null, 'failed_rate' => null, 'failed_avg_time' => null];
		$connection = $manager->connection($conn);
		if ($connection instanceof Illuminate\Queue\DatabaseQueue) {
			$sizes[$conn][$q]['driver'] = "database";
			try {
				$db = $connection->getDatabase();
				$reflection = new ReflectionClass($connection);
				$property = $reflection->getProperty('table');
				$property->setAccessible(true);
				$table = $property->getValue($connection);
				$oldestPending = $db->table($table)->where("queue", $q)->whereNull("reserved_at")->orderBy("created_at")->value("created_at");

				$sizes[$conn][$q]['pending'] = $db->table($table)->where("queue", $q)->whereNull("reserved_at")->where("available_at", "<=", $now->timestamp)->count();
				$sizes[$conn][$q]['scheduled'] = $db->table($table)->where("queue", $q)->where("available_at", ">", $now->timestamp)->count();
				$sizes[$conn][$q]['reserved'] = $db->table($table)->where("queue", $q)->whereNotNull("reserved_at")->count();
				$sizes[$conn][$q]['oldest_pending'] = $oldestPending ? (int) now()->diffInSeconds(Carbon::createFromTimestamp($oldestPending), true) : null;
			} catch (\Throwable $e) {
				$sizes[$conn][$q]['error'] = $e->getMessage();
			}
		}
		if ($connection instanceof Illuminate\Queue\RedisQueue) {
			$sizes[$conn][$q]['driver'] = "redis";
			try {
				$redis = $connection->getConnection();
				$queueKey = $connection->getQueue($q);
	
				$sizes[$conn][$q]['size'] = $redis->llen($queueKey);
				$sizes[$conn][$q]['pending'] = $redis->llen($queueKey);
				$sizes[$conn][$q]['scheduled'] = $redis->zcard($queueKey.':delayed');
				$sizes[$conn][$q]['reserved'] = $redis->zcard($queueKey.':reserved');
	
				$oldestRaw = $redis->lindex($queueKey, 0);
				if ($oldestRaw) {
					$decoded = json_decode($oldestRaw, true);
					if (isset($decoded['createdAt'])) {
						$sizes[$conn][$q]['oldest_pending'] = $decoded['createdAt'] ? (int) Carbon::createFromTimestamp($decoded['createdAt'])->diffInSeconds($now, true) : null;
					}
				}
			} catch (\Throwable $e) {
				$sizes[$conn][$q]['error'] = $e->getMessage();
			}
		}
		$sizes[$conn][$q]['size'] = $manager->connection($conn)->size($q);

		try {
			if ($failedJobsProvider instanceof Illuminate\Queue\Failed\DatabaseFailedJobProvider
				|| $failedJobsProvider instanceof Illuminate\Queue\Failed\DatabaseUuidFailedJobProvider) {
			
				$failedProviderReflection = new ReflectionClass($failedJobsProvider);
				$method = $failedProviderReflection->getMethod('getTable');
				$method->setAccessible(true);
				$query = $method->invoke($failedJobsProvider);

				$minutes = [1, 5, 10];
				$failed = [];
				$failedRates = [];
			
				$baseQuery = $query->where('connection', $conn)->where('queue', $q);;
	
				foreach ($minutes as $min) {
					$from = now()->subMinutes($min);
			
					$count = (clone $baseQuery)
						->where('failed_at', '>=', $from)
						->count();
			
					$failed[$min] = $count;
					$failedRates[$min] = round($count / $min, 2);
				}
	
				$oldestFailed = (clone $baseQuery)
					->whereNotNull('failed_at')
					->orderBy('failed_at', 'asc')
					->value('failed_at');
			
				$newestFailed = (clone $baseQuery)
					->whereNotNull('failed_at')
					->orderBy('failed_at', 'desc')
					->value('failed_at');
			
				$sizes[$conn][$q]['failed'] = (clone $baseQuery)->count() ?? null;
				$sizes[$conn][$q]['failed_rate_1m'] = $failedRates[1] ?? null;
				$sizes[$conn][$q]['failed_rate_5m'] = $failedRates[5] ?? null;
				$sizes[$conn][$q]['failed_rate_10m'] = $failedRates[10] ?? null;
				$sizes[$conn][$q]['failed_1m'] = $failed[1] ?? null;
				$sizes[$conn][$q]['failed_5m'] = $failed[5] ?? null;
				$sizes[$conn][$q]['failed_10m'] = $failed[10] ?? null;
				$sizes[$conn][$q]['oldest_failed'] = $oldestFailed ? (int) Carbon::parse($oldestFailed)->diffInSeconds($now, true) : null;
				$sizes[$conn][$q]['newest_failed'] = $newestFailed ? (int) Carbon::parse($newestFailed)->diffInSeconds($now, true) : null;
			
			} else {
				$sizes[$conn][$q]['error'] = "Unknown class ". $failedJobsProvider;
			}
		} catch (\Throwable $e) {
			$sizes[$conn][$q]['error'] = $e->getMessage();
		}
		
	} catch (\Throwable $e) {
		$sizes[$conn][$q]['size'] = null;
		$sizes[$conn][$q]['error'] = $e->getMessage();
	}
}
}`

	script += `

echo json_encode($sizes);`

	return script, nil
}

func runQueueScript(ctx context.Context, appPath string, phpBinary string, script string) (*QueueSizes, error) {
	cmd := exec.CommandContext(ctx, phpBinary, "-d", "error_reporting=E_ALL & ~E_DEPRECATED", "artisan", "tinker", "--execute", script)
	cmd.Dir = filepath.Clean(appPath)

	// disable monitoring on scraping to prevent exhausting monitoring tools
	cmd.Env = os.Environ()
	cmd.Env = append(cmd.Env, "NIGHTWATCH_ENABLED=false")
	cmd.Env = append(cmd.Env, "TELESCOPE_ENABLED=false")
	cmd.Env = append(cmd.Env, "NEW_RELIC_ENABLED=false")
	cmd.Env = append(cmd.Env, "BUGSNAG_API_KEY=null")
	cmd.Env = append(cmd.Env, "SENTRY_LARAVEL_DSN=null")
	cmd.Env = append(cmd.Env, "ROLLBAR_TOKEN=null")

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	if err := cmd.Run(); err != nil {
		if ctx.Err() != nil {
			return nil, fmt.Errorf("artisan tinker timed out: %w", ctx.Err())
		}
		return nil, fmt.Errorf("artisan tinker failed: %w\nOutput: %s", err, out.String())
	}

	result := QueueSizes{}
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("failed to parse output: %w\nOutput: %s", err, out.String())
	}

	return &result, nil
}
