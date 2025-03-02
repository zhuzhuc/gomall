module github.com/bytedance-youthcamp/demo

go 1.24

toolchain go1.24.0

replace (
	github.com/bytedance-youthcamp/demo/api/cart => ./api/cart
	github.com/bytedance-youthcamp/demo/api/order => ./api/order
	github.com/bytedance-youthcamp/demo/api/product => ./api/product
	github.com/bytedance-youthcamp/demo/api/user => ./api/user
	github.com/bytedance-youthcamp/demo/api/userapi => ./api/userapi
	github.com/bytedance-youthcamp/demo/internal/api/userapi => ./internal/api/userapi
	github.com/bytedance-youthcamp/demo/internal/config => ./internal/config
	github.com/bytedance-youthcamp/demo/internal/repository => ./internal/repository
	github.com/bytedance-youthcamp/demo/internal/service/auth => ./internal/service/auth
	github.com/bytedance-youthcamp/demo/internal/service/order => ./internal/service/order
	github.com/bytedance-youthcamp/demo/internal/service/payment => ./internal/service/payment
	github.com/bytedance-youthcamp/demo/internal/service/product => ./internal/service/product
)

require (
	github.com/bytedance-youthcamp/demo/api/cart v0.0.0
	github.com/bytedance-youthcamp/demo/api/product v0.0.0
	github.com/bytedance-youthcamp/demo/api/user v0.0.0
	github.com/bytedance-youthcamp/demo/internal/config v0.0.0
	github.com/bytedance-youthcamp/demo/internal/service/auth v0.0.0
	github.com/go-sql-driver/mysql v1.8.1
	github.com/google/uuid v1.6.0
	github.com/lib/pq v1.10.9
	github.com/mattn/go-sqlite3 v1.14.24
	github.com/pquerna/otp v1.4.0
	github.com/prometheus/client_golang v1.21.0
	github.com/spf13/viper v1.19.0
	github.com/stretchr/testify v1.10.0
	golang.org/x/crypto v0.32.0
	google.golang.org/grpc v1.64.0
	google.golang.org/protobuf v1.36.5
	gorm.io/driver/mysql v1.5.7
	gorm.io/gorm v1.25.12
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/boombuler/barcode v1.0.1-0.20190219062509-6c824513bacc // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/golang-jwt/jwt/v5 v5.2.1 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.62.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/stretchr/objx v0.5.2 // indirect
	go.etcd.io/etcd/client/v3 v3.5.18 // indirect
)

require (
	github.com/coreos/go-semver v0.3.0 // indirect
	github.com/coreos/go-systemd/v22 v22.3.2 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.etcd.io/etcd/api/v3 v3.5.18 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.18 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	go.uber.org/zap v1.27.0
	golang.org/x/exp v0.0.0-20230905200255-921286631fa9 // indirect
	golang.org/x/net v0.34.0 // indirect
	golang.org/x/sys v0.29.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240318140521-94a12d6c2237 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240318140521-94a12d6c2237 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
