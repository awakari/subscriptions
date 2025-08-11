package config

import (
	"github.com/kelseyhightower/envconfig"
	"time"
)

type Config struct {
	Api struct {
		Port    uint16 `envconfig:"API_PORT" default:"50051" required:"true"`
		Auth    AuthConfig
		Events  EventsConfig
		Http    HttpConfig
		Queue   QueueConfig
		Usage   UsageConfig
		Metrics MetricsConfig
	}
	Db  DbConfig
	Log struct {
		Level int `envconfig:"LOG_LEVEL" default:"-4" required:"true"`
	}
}

type AuthConfig struct {
	Uri string `envconfig:"API_AUTH_URI" default:"auth:50051" required:"true"`
}

type QueueConfig struct {
	BackoffError time.Duration `envconfig:"API_QUEUE_BACKOFF_ERROR" default:"1s" required:"true"`
	Uri          string        `envconfig:"API_QUEUE_URI" default:"queue:50051" required:"true"`
	Connection   struct {
		Count struct {
			Init uint32 `envconfig:"API_QUEUE_CONN_COUNT_INIT" default:"1" required:"true"`
			Max  uint32 `envconfig:"API_QUEUE_CONN_COUNT_MAX" default:"5" required:"true"`
		}
		IdleTimeout time.Duration `envconfig:"API_QUEUE_CONN_IDLE_TIMEOUT" default:"15m" required:"true"`
	}
	InterestDeleted struct {
		BatchSize uint32 `envconfig:"API_QUEUE_INTEREST_DELETED_BATCH_SIZE" default:"10" required:"true"`
		Name      string `envconfig:"API_QUEUE_INTEREST_DELETED_NAME" default:"reader" required:"true"`
		Subj      string `envconfig:"API_QUEUE_INTEREST_DELETED_SUBJ" default:"interests-deleted" required:"true"`
	}
}

type UsageConfig struct {
	Uri        string `envconfig:"API_USAGE_URI" default:"usage:50051" required:"true"`
	Connection struct {
		Count struct {
			Init uint32 `envconfig:"API_USAGE_CONN_COUNT_INIT" default:"1" required:"true"`
			Max  uint32 `envconfig:"API_USAGE_CONN_COUNT_MAX" default:"2" required:"true"`
		}
		IdleTimeout time.Duration `envconfig:"API_USAGE_CONN_IDLE_TIMEOUT" default:"15m" required:"true"`
	}
}

type DbConfig struct {
	Uri      string `envconfig:"DB_URI" default:"mongodb://localhost:27017/?retryWrites=true&w=majority" required:"true"`
	Name     string `envconfig:"DB_NAME" default:"subscriptions" required:"true"`
	UserName string `envconfig:"DB_USERNAME" default:""`
	Password string `envconfig:"DB_PASSWORD" default:""`
	Table    struct {
		Name      string        `envconfig:"DB_TABLE_NAME" default:"subscriptions" required:"true"`
		Retention time.Duration `envconfig:"DB_TABLE_RETENTION" default:"2160h" required:"true"`
		Shard     bool          `envconfig:"DB_TABLE_SHARD" default:"true"`
	}
	Tls struct {
		Enabled  bool `envconfig:"DB_TLS_ENABLED" default:"false" required:"true"`
		Insecure bool `envconfig:"DB_TLS_INSECURE" default:"false" required:"true"`
	}
	Cache CacheConfig
}

type HttpConfig struct {
	Enabled   bool   `envconfig:"API_HTTP_ENABLED" default:"true"`
	Host      string `envconfig:"API_HTTP_HOST" default:"http://reader.local" required:"true"`
	Port      uint16 `envconfig:"API_HTTP_PORT" default:"8080" required:"true"`
	UserAgent string `envconfig:"API_HTTP_USER_AGENT" default:"awakari" required:"true"`
}

type CacheConfig struct {
	Local struct {
		Size uint32 `envconfig:"DB_CACHE_LOCAL_SIZE" default:"1024" required:"true"`
	}
	Ttl      time.Duration `envconfig:"DB_CACHE_TTL" default:"1h" required:"true"`
	Addr     string        `envconfig:"DB_CACHE_ADDR" default:"cache-keydb:6379" required:"true"`
	Password string        `envconfig:"DB_CACHE_PASSWORD" required:"false"`
}

type EventsConfig struct {
	Uri        string `envconfig:"API_EVENTS_URI" default:"events:50051" required:"true"`
	Connection struct {
		Count struct {
			Init uint32 `envconfig:"API_EVENTS_CONN_COUNT_INIT" default:"1" required:"true"`
			Max  uint32 `envconfig:"API_EVENTS_CONN_COUNT_MAX" default:"10" required:"true"`
		}
		IdleTimeout time.Duration `envconfig:"API_EVENTS_CONN_IDLE_TIMEOUT" default:"15m" required:"true"`
	}
	FollowersChanged struct {
		Source string `envconfig:"API_EVENTS_FOLLOWERS_CHANGED_SOURCE" default:"https://awakari.com/reader" required:"true"`
		Limit  uint32 `envconfig:"API_EVENTS_FOLLOWERS_CHANGED_LIMIT" default:"1000" required:"true"`
		Topic  string `envconfig:"API_EVENTS_FOLLOWERS_CHANGED_TOPIC" default:"followers-changed" required:"true"`
	}
}

type MetricsConfig struct {
	Port uint16 `envconfig:"API_METRICS_PORT" default:"9090" required:"true"`
}

func NewConfigFromEnv() (cfg Config, err error) {
	err = envconfig.Process("", &cfg)
	return
}
