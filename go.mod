// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

module github.com/apache/dubbo-kubernetes

go 1.23.0

require (
	dubbo.apache.org/dubbo-go/v3 v3.2.0-rc1.0.20240926033233-f03922ab1bcb
	github.com/AlecAivazis/survey/v2 v2.3.7
	github.com/Masterminds/semver/v3 v3.2.1
	github.com/asaskevich/govalidator v0.0.0-20230301143203-a9d515a09cc2
	github.com/bakito/go-log-logr-adapter v0.0.2
	github.com/buildpacks/pack v0.30.0
	github.com/cheggaaa/pb/v3 v3.1.5
	github.com/containers/image/v5 v5.34.0
	github.com/containers/storage v1.57.1
	github.com/docker/cli v27.5.1+incompatible
	github.com/docker/docker v27.5.1+incompatible
	github.com/docker/docker-credential-helpers v0.8.2
	github.com/docker/go-connections v0.5.0
	github.com/dubbogo/go-zookeeper v1.0.4-0.20211212162352-f9d2183d89d5
	github.com/dubbogo/gost v1.14.0
	github.com/emicklei/go-restful/v3 v3.11.0
	github.com/envoyproxy/go-control-plane v0.13.1
	github.com/envoyproxy/protoc-gen-validate v1.1.0
	github.com/evanphx/json-patch v5.7.0+incompatible
	github.com/fatih/color v1.18.0
	github.com/fullstorydev/grpcurl v1.9.1
	github.com/ghodss/yaml v1.0.0
	github.com/gin-gonic/gin v1.9.1
	github.com/go-co-op/gocron v1.9.0
	github.com/go-git/go-billy/v5 v5.6.2
	github.com/go-git/go-git/v5 v5.13.1
	github.com/go-logr/logr v1.4.2
	github.com/go-logr/zapr v1.3.0
	github.com/go-sql-driver/mysql v1.7.0
	github.com/goburrow/cache v0.1.4
	github.com/gogo/protobuf v1.3.2
	github.com/golang/mock v1.6.0
	github.com/golang/protobuf v1.5.4
	github.com/google/go-cmp v0.6.0
	github.com/google/go-containerregistry v0.20.2
	github.com/google/uuid v1.6.0
	github.com/grpc-ecosystem/go-grpc-middleware v1.3.0
	github.com/hashicorp/go-multierror v1.1.1
	github.com/heroku/color v0.0.6
	github.com/hoisie/mustache v0.0.0-20160804235033-6375acf62c69
	github.com/jhump/protoreflect v1.16.0
	github.com/kelseyhightower/envconfig v1.4.0
	github.com/mitchellh/mapstructure v1.5.0
	github.com/moby/term v0.5.0
	github.com/onsi/ginkgo/v2 v2.22.1
	github.com/onsi/gomega v1.36.2
	github.com/ory/viper v1.7.5
	github.com/patrickmn/go-cache v2.1.0+incompatible
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v1.20.5
	github.com/sethvargo/go-retry v0.2.4
	github.com/slok/go-http-metrics v0.11.0
	github.com/spf13/cobra v1.8.1
	github.com/spf13/pflag v1.0.5
	github.com/spiffe/go-spiffe/v2 v2.1.7
	github.com/stretchr/testify v1.10.0
	go.opentelemetry.io/otel/trace v1.31.0
	go.uber.org/atomic v1.10.0
	go.uber.org/multierr v1.11.0
	go.uber.org/zap v1.27.0
	golang.org/x/crypto v0.32.0
	golang.org/x/exp v0.0.0-20241217172543-b2144cdd0a67
	golang.org/x/net v0.34.0
	golang.org/x/sync v0.10.0
	golang.org/x/sys v0.29.0
	golang.org/x/term v0.28.0
	golang.org/x/text v0.21.0
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250102185135-69823020774d
	google.golang.org/grpc v1.69.4
	google.golang.org/protobuf v1.36.2
	gopkg.in/natefinch/lumberjack.v2 v2.2.1
	gopkg.in/yaml.v3 v3.0.1
	gorm.io/driver/mysql v1.5.4
	gorm.io/driver/sqlite v1.5.5
	gorm.io/gorm v1.25.7
	helm.sh/helm/v3 v3.12.3
	k8s.io/api v0.32.0
	k8s.io/apiextensions-apiserver v0.32.0
	k8s.io/apimachinery v0.32.0
	k8s.io/client-go v0.32.0
	k8s.io/kube-openapi v0.0.0-20241105132330-32ad38e42d3f
	k8s.io/kubectl v0.27.3
	k8s.io/utils v0.0.0-20241104100929-3ea5e8cea738
	sigs.k8s.io/controller-runtime v0.19.4
	sigs.k8s.io/controller-tools v0.17.0
	sigs.k8s.io/yaml v1.4.0
)

require (
	cel.dev/expr v0.18.0 // indirect
	cloud.google.com/go/compute/metadata v0.5.2 // indirect
	dario.cat/mergo v1.0.1 // indirect
	github.com/Azure/azure-sdk-for-go v68.0.0+incompatible // indirect
	github.com/Azure/go-ansiterm v0.0.0-20230124172434-306776ec8161 // indirect
	github.com/Azure/go-autorest v14.2.0+incompatible // indirect
	github.com/Azure/go-autorest/autorest v0.11.29 // indirect
	github.com/Azure/go-autorest/autorest/adal v0.9.23 // indirect
	github.com/Azure/go-autorest/autorest/azure/auth v0.5.12 // indirect
	github.com/Azure/go-autorest/autorest/azure/cli v0.4.6 // indirect
	github.com/Azure/go-autorest/autorest/date v0.3.0 // indirect
	github.com/Azure/go-autorest/logger v0.2.1 // indirect
	github.com/Azure/go-autorest/tracing v0.6.0 // indirect
	github.com/BurntSushi/toml v1.4.0 // indirect
	github.com/Masterminds/goutils v1.1.1 // indirect
	github.com/Masterminds/semver v1.5.0 // indirect
	github.com/Masterminds/sprig/v3 v3.2.3 // indirect
	github.com/Microsoft/go-winio v0.6.2 // indirect
	github.com/OneOfOne/xxhash v1.2.8 // indirect
	github.com/ProtonMail/go-crypto v1.1.3 // indirect
	github.com/RoaringBitmap/roaring v1.2.3 // indirect
	github.com/VividCortex/ewma v1.2.0 // indirect
	github.com/Workiva/go-datastructures v1.0.52 // indirect
	github.com/afex/hystrix-go v0.0.0-20180502004556-fa1af6a1f4f5 // indirect
	github.com/agext/levenshtein v1.2.3 // indirect
	github.com/alibaba/sentinel-golang v1.0.4 // indirect
	github.com/aliyun/alibaba-cloud-sdk-go v1.61.1800 // indirect
	github.com/apache/dubbo-getty v1.4.10 // indirect
	github.com/apache/dubbo-go-hessian2 v1.12.2 // indirect
	github.com/apex/log v1.9.0 // indirect
	github.com/aws/aws-sdk-go-v2 v1.18.1 // indirect
	github.com/aws/aws-sdk-go-v2/config v1.18.27 // indirect
	github.com/aws/aws-sdk-go-v2/credentials v1.13.26 // indirect
	github.com/aws/aws-sdk-go-v2/feature/ec2/imds v1.13.4 // indirect
	github.com/aws/aws-sdk-go-v2/internal/configsources v1.1.34 // indirect
	github.com/aws/aws-sdk-go-v2/internal/endpoints/v2 v2.4.28 // indirect
	github.com/aws/aws-sdk-go-v2/internal/ini v1.3.35 // indirect
	github.com/aws/aws-sdk-go-v2/service/ecr v1.18.11 // indirect
	github.com/aws/aws-sdk-go-v2/service/ecrpublic v1.16.2 // indirect
	github.com/aws/aws-sdk-go-v2/service/internal/presigned-url v1.9.28 // indirect
	github.com/aws/aws-sdk-go-v2/service/sso v1.12.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/ssooidc v1.14.12 // indirect
	github.com/aws/aws-sdk-go-v2/service/sts v1.19.2 // indirect
	github.com/aws/smithy-go v1.13.5 // indirect
	github.com/awslabs/amazon-ecr-credential-helper/ecr-login v0.0.0-20230522190001-adf1bafd791a // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/bits-and-blooms/bitset v1.2.0 // indirect
	github.com/blang/semver/v4 v4.0.0 // indirect
	github.com/bufbuild/protocompile v0.10.0 // indirect
	github.com/buger/jsonparser v1.1.1 // indirect
	github.com/buildpacks/imgutil v0.0.0-20230626185301-726f02e4225c // indirect
	github.com/buildpacks/lifecycle v0.17.0 // indirect
	github.com/bytedance/sonic v1.9.1 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/census-instrumentation/opencensus-proto v0.4.1 // indirect
	github.com/cespare/xxhash v1.1.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/chenzhuoyu/base64x v0.0.0-20221115062448-fe3a3abad311 // indirect
	github.com/chrismellard/docker-credential-acr-env v0.0.0-20230304212654-82a0ddb27589 // indirect
	github.com/cloudflare/circl v1.3.7 // indirect
	github.com/cncf/udpa/go v0.0.0-20220112060539-c52dc94e7fbe // indirect
	github.com/cncf/xds/go v0.0.0-20240905190251-b4127c9b8d78 // indirect
	github.com/containerd/containerd v1.7.2 // indirect
	github.com/containerd/stargz-snapshotter/estargz v0.16.3 // indirect
	github.com/containerd/typeurl v1.0.2 // indirect
	github.com/coreos/go-semver v0.3.1 // indirect
	github.com/coreos/go-systemd/v22 v22.5.0 // indirect
	github.com/creasty/defaults v1.5.2 // indirect
	github.com/cyphar/filepath-securejoin v0.3.6 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/dgraph-io/ristretto v0.0.1 // indirect
	github.com/dimchansky/utfbom v1.1.1 // indirect
	github.com/dlclark/regexp2 v1.7.0 // indirect
	github.com/docker/distribution v2.8.3+incompatible // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/dop251/goja v0.0.0-20240220182346-e401ed450204 // indirect
	github.com/dubbogo/grpc-go v1.42.10 // indirect
	github.com/dubbogo/triple v1.2.2-rc3 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/evanphx/json-patch/v5 v5.9.0 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/fxamacker/cbor/v2 v2.7.0 // indirect
	github.com/gabriel-vasile/mimetype v1.4.2 // indirect
	github.com/gdamore/encoding v1.0.0 // indirect
	github.com/gdamore/tcell/v2 v2.6.0 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-errors/errors v1.4.2 // indirect
	github.com/go-git/gcfg v1.5.1-0.20230307220236-3a3c6141e376 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.2.6 // indirect
	github.com/go-openapi/jsonpointer v0.21.0 // indirect
	github.com/go-openapi/jsonreference v0.21.0 // indirect
	github.com/go-openapi/swag v0.23.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/go-playground/validator/v10 v10.15.5 // indirect
	github.com/go-resty/resty/v2 v2.7.0 // indirect
	github.com/go-sourcemap/sourcemap v2.1.3+incompatible // indirect
	github.com/go-task/slim-sprig/v3 v3.0.0 // indirect
	github.com/gobuffalo/flect v1.0.3 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/goccy/go-json v0.10.2 // indirect
	github.com/golang-jwt/jwt/v4 v4.5.0 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/btree v1.0.1 // indirect
	github.com/google/gnostic-models v0.6.9 // indirect
	github.com/google/gofuzz v1.2.0 // indirect
	github.com/google/pprof v0.0.0-20241210010833-40e02aabc2ad // indirect
	github.com/google/shlex v0.0.0-20191202100458-e7afc7fbc510 // indirect
	github.com/gorilla/websocket v1.5.0 // indirect
	github.com/gregjones/httpcache v0.0.0-20190611155906-901d90724c79 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.20.0 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/golang-lru v1.0.2 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/hashicorp/vault/sdk v0.7.0 // indirect
	github.com/huandu/xstrings v1.4.0 // indirect
	github.com/imdario/mergo v0.3.16 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/influxdata/tdigest v0.0.1 // indirect
	github.com/jbenet/go-context v0.0.0-20150711004518-d14ea06fba99 // indirect
	github.com/jinzhu/copier v0.3.5 // indirect
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/jmespath/go-jmespath v0.4.0 // indirect
	github.com/josharian/intern v1.0.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/k0kubun/pp v3.0.1+incompatible // indirect
	github.com/kballard/go-shellquote v0.0.0-20180428030007-95032a82bc51 // indirect
	github.com/kevinburke/ssh_config v1.2.0 // indirect
	github.com/klauspost/compress v1.17.11 // indirect
	github.com/klauspost/cpuid/v2 v2.2.4 // indirect
	github.com/klauspost/pgzip v1.2.6 // indirect
	github.com/knadh/koanf v1.5.0 // indirect
	github.com/leodido/go-urn v1.2.4 // indirect
	github.com/lucasb-eyer/go-colorful v1.2.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20211012122336-39d0f177ccd0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mailru/easyjson v0.7.7 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/mattn/go-sqlite3 v1.14.24 // indirect
	github.com/mgutz/ansi v0.0.0-20170206155736-9520e82c474b // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/go-homedir v1.1.0 // indirect
	github.com/mitchellh/ioprogress v0.0.0-20180201004757-6a23b12fa88e // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/buildkit v0.11.6 // indirect
	github.com/moby/patternmatcher v0.5.0 // indirect
	github.com/moby/sys/capability v0.4.0 // indirect
	github.com/moby/sys/mountinfo v0.7.2 // indirect
	github.com/moby/sys/sequential v0.5.0 // indirect
	github.com/moby/sys/user v0.3.0 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/monochromegane/go-gitignore v0.0.0-20200626010858-205db1a8cc00 // indirect
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/mschoch/smat v0.2.0 // indirect
	github.com/munnerz/goautoneg v0.0.0-20191010083416-a7dc8b61c822 // indirect
	github.com/nacos-group/nacos-sdk-go/v2 v2.2.2 // indirect
	github.com/natefinch/lumberjack v2.0.0+incompatible // indirect
	github.com/oliveagle/jsonpath v0.0.0-20180606110733-2e52cf6e6852 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0 // indirect
	github.com/opencontainers/runc v1.1.7 // indirect
	github.com/opencontainers/runtime-spec v1.2.0 // indirect
	github.com/opencontainers/selinux v1.11.1 // indirect
	github.com/opentracing/opentracing-go v1.2.0 // indirect
	github.com/openzipkin/zipkin-go v0.4.0 // indirect
	github.com/pelletier/go-toml v1.9.5 // indirect
	github.com/pelletier/go-toml/v2 v2.1.1 // indirect
	github.com/peterbourgon/diskv v2.0.1+incompatible // indirect
	github.com/pierrec/lz4 v2.6.1+incompatible // indirect
	github.com/pjbgf/sha1cd v0.3.0 // indirect
	github.com/planetscale/vtprotobuf v0.6.1-0.20240319094008-0393e58bdf10 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/polarismesh/polaris-go v1.3.0 // indirect
	github.com/power-devops/perfstat v0.0.0-20210106213030-5aafc221ea8c // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.57.0 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/rivo/tview v0.0.0-20220307222120-9994674d60a8 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/robfig/cron/v3 v3.0.1 // indirect
	github.com/sabhiram/go-gitignore v0.0.0-20210923224102-525f6e181f06 // indirect
	github.com/sergi/go-diff v1.3.2-0.20230802210424-5b0b94c5c0d3 // indirect
	github.com/shirou/gopsutil/v3 v3.22.2 // indirect
	github.com/shopspring/decimal v1.3.1 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/skeema/knownhosts v1.3.0 // indirect
	github.com/spaolacci/murmur3 v1.1.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/tklauser/go-sysconf v0.3.10 // indirect
	github.com/tklauser/numcpus v0.4.0 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/uber/jaeger-client-go v2.29.1+incompatible // indirect
	github.com/uber/jaeger-lib v2.4.1+incompatible // indirect
	github.com/ugorji/go/codec v1.2.11 // indirect
	github.com/ulikunitz/xz v0.5.12 // indirect
	github.com/vbatts/tar-split v0.11.7 // indirect
	github.com/x448/float16 v0.8.4 // indirect
	github.com/xanzy/ssh-agent v0.3.3 // indirect
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonreference v0.0.0-20180127040603-bd5ef7bd5415 // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	github.com/xlab/treeprint v1.2.0 // indirect
	github.com/yusufpapurcu/wmi v1.2.2 // indirect
	go.etcd.io/etcd/api/v3 v3.5.16 // indirect
	go.etcd.io/etcd/client/pkg/v3 v3.5.16 // indirect
	go.etcd.io/etcd/client/v3 v3.5.16 // indirect
	go.opentelemetry.io/contrib/propagators/b3 v1.10.0 // indirect
	go.opentelemetry.io/otel v1.31.0 // indirect
	go.opentelemetry.io/otel/exporters/jaeger v1.17.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.28.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.27.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.28.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.10.0 // indirect
	go.opentelemetry.io/otel/exporters/zipkin v1.10.0 // indirect
	go.opentelemetry.io/otel/metric v1.31.0 // indirect
	go.opentelemetry.io/otel/sdk v1.31.0 // indirect
	go.opentelemetry.io/proto/otlp v1.3.1 // indirect
	golang.org/x/arch v0.3.0 // indirect
	golang.org/x/mod v0.22.0 // indirect
	golang.org/x/oauth2 v0.25.0 // indirect
	golang.org/x/time v0.7.0 // indirect
	golang.org/x/tools v0.28.0 // indirect
	gomodules.xyz/jsonpatch/v2 v2.4.0 // indirect
	google.golang.org/genproto v0.0.0-20240213162025-012b6fc9bca9 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20241015192408-796eee8c2d53 // indirect
	gopkg.in/evanphx/json-patch.v4 v4.12.0 // indirect
	gopkg.in/inf.v0 v0.9.1 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/warnings.v0 v0.1.2 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	k8s.io/cli-runtime v0.32.0 // indirect
	k8s.io/klog/v2 v2.130.1 // indirect
	sigs.k8s.io/json v0.0.0-20241010143419-9aa6b5e7a4b3 // indirect
	sigs.k8s.io/kustomize/api v0.18.0 // indirect
	sigs.k8s.io/kustomize/kyaml v0.18.1 // indirect
	sigs.k8s.io/structured-merge-diff/v4 v4.4.2 // indirect
)

replace (
	github.com/docker/cli => github.com/docker/cli v23.0.1+incompatible
	github.com/docker/distribution => github.com/docker/distribution v2.8.2+incompatible
	github.com/docker/docker => github.com/docker/docker v23.0.1+incompatible
)
