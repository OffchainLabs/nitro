load("@bazel_tools//tools/build_defs/repo:http.bzl", "http_archive")

http_archive(
    name = "io_bazel_rules_go",
    sha256 = "ae013bf35bd23234d1dea46b079f1e05ba74ac0321423830119d3e787ec73483",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/rules_go/releases/download/v0.36.0/rules_go-v0.36.0.zip",
        "https://github.com/bazelbuild/rules_go/releases/download/v0.36.0/rules_go-v0.36.0.zip",
    ],
)

http_archive(
    name = "bazel_gazelle",
    sha256 = "448e37e0dbf61d6fa8f00aaa12d191745e14f07c31cabfa731f0c8e8a4f41b97",
    urls = [
        "https://mirror.bazel.build/github.com/bazelbuild/bazel-gazelle/releases/download/v0.28.0/bazel-gazelle-v0.28.0.tar.gz",
        "https://github.com/bazelbuild/bazel-gazelle/releases/download/v0.28.0/bazel-gazelle-v0.28.0.tar.gz",
    ],
)

load("@io_bazel_rules_go//go:deps.bzl", "go_register_toolchains", "go_rules_dependencies")
load("@bazel_gazelle//:deps.bzl", "gazelle_dependencies", "go_repository")
load("//:deps.bzl", "go_dependencies")

go_repository(
    name = "com_github_emicklei_dot",
    importpath = "github.com/emicklei/dot",
    sum = "h1:WjL422LPltH/ThM9AJQ8HJXEMw9SOxLrglppg/1pFYU=",
    version = "v1.2.0",
)

go_repository(
    name = "com_github_labstack_echo_v5",
    importpath = "github.com/labstack/echo/v5",
    sum = "h1:07/Vl+8iDqHkOFA59B2UvXfV/Wih+YHoEaLTWSxRiRE=",
    version = "v5.0.0-20220717203827-74022662be4a",
)

go_repository(
    name = "com_github_valyala_bytebufferpool",
    importpath = "github.com/valyala/bytebufferpool",
    sum = "h1:GqA5TC/0021Y/b9FG4Oi9Mr3q7XYx6KllzawFIhcdPw=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_valyala_fasttemplate",
    importpath = "github.com/valyala/fasttemplate",
    sum = "h1:TVEnxayobAdVkhQfrfes2IzOB6o+z4roRkPF52WA1u4=",
    version = "v1.2.1",
)

go_repository(
    name = "com_github_aristanetworks_goarista",
    importpath = "github.com/aristanetworks/goarista",
    sum = "h1:XJH0YfVFKbq782tlNThzN/Ud5qm/cx6LXOA/P6RkTxc=",
    version = "v0.0.0-20200805130819-fd197cf57d96",
)

go_repository(
    name = "com_github_bazelbuild_rules_go",
    importpath = "github.com/bazelbuild/rules_go",
    sum = "h1:Wxu7JjqnF78cKZbsBsARLSXx/jlGaSLCnUV3mTlyHvM=",
    version = "v0.23.2",
)

go_repository(
    name = "com_github_benbjohnson_clock",
    importpath = "github.com/benbjohnson/clock",
    sum = "h1:ip6w0uFQkncKQ979AypyG0ER7mqUSBdKLOgAle/AT8A=",
    version = "v1.3.0",
)

go_repository(
    name = "com_github_beorn7_perks",
    importpath = "github.com/beorn7/perks",
    sum = "h1:VlbKKnNfV8bJzeqoa4cOKqO6bYr3WgKZxO8Z16+hsOM=",
    version = "v1.0.1",
)

go_repository(
    name = "com_github_burntsushi_toml",
    importpath = "github.com/BurntSushi/toml",
    sum = "h1:ksErzDEI1khOiGPgpwuI7x2ebx/uXQNw7xJpn9Eq1+I=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_cespare_xxhash",
    importpath = "github.com/cespare/xxhash",
    sum = "h1:a6HrQnmkObjyL+Gs60czilIUGqrzKutQD6XZog3p+ko=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_chzyer_readline",
    importpath = "github.com/chzyer/readline",
    sum = "h1:lSwwFrbNviGePhkewF1az4oLmcwqCZijQ2/Wi3BGHAI=",
    version = "v1.5.0",
)

go_repository(
    name = "com_github_containerd_cgroups",
    importpath = "github.com/containerd/cgroups",
    sum = "h1:jN/mbWBEaz+T1pi5OFtnkQ+8qnmEbAr1Oo1FRm5B0dA=",
    version = "v1.0.4",
)

go_repository(
    name = "com_github_coreos_go_systemd",
    importpath = "github.com/coreos/go-systemd",
    sum = "h1:iW4rZ826su+pqaw19uhpSCzhj44qo35pNgKFGqzDKkU=",
    version = "v0.0.0-20191104093116-d3cd4ed1dbcf",
)

go_repository(
    name = "com_github_coreos_go_systemd_v22",
    importpath = "github.com/coreos/go-systemd/v22",
    sum = "h1:RrqgGjYQKalulkV8NGVIfkXQf6YYmOyiJKk8iXXhfZs=",
    version = "v22.5.0",
)

go_repository(
    name = "com_github_d4l3k_messagediff",
    importpath = "github.com/d4l3k/messagediff",
    sum = "h1:ZcAIMYsUg0EAp9X+tt8/enBE/Q8Yd5kzPynLyKptt9U=",
    version = "v1.2.1",
)

go_repository(
    name = "com_github_davidlazar_go_crypto",
    importpath = "github.com/davidlazar/go-crypto",
    sum = "h1:pFUpOrbxDR6AkioZ1ySsx5yxlDQZ8stG2b88gTPxgJU=",
    version = "v0.0.0-20200604182044-b73af7476f6c",
)

go_repository(
    name = "com_github_dgraph_io_ristretto",
    importpath = "github.com/dgraph-io/ristretto",
    sum = "h1:cNcG4c2n5xanQzp2hMyxDxPYVQmZ91y4WN6fJFlndLo=",
    version = "v0.0.4-0.20210318174700-74754f61e018",
)

go_repository(
    name = "com_github_docker_go_units",
    importpath = "github.com/docker/go-units",
    sum = "h1:69rxXcBk27SvSaaxTtLh/8llcHD8vYHT7WSdRZ/jvr4=",
    version = "v0.5.0",
)

go_repository(
    name = "com_github_dustin_go_humanize",
    importpath = "github.com/dustin/go-humanize",
    sum = "h1:VSnTsYCnlFHaM2/igO1h6X3HA71jcobQuxemgkq4zYo=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_elastic_gosigar",
    importpath = "github.com/elastic/gosigar",
    sum = "h1:Dg80n8cr90OZ7x+bAax/QjoW/XqTI11RmA79ZwIm9/4=",
    version = "v0.14.2",
)

go_repository(
    name = "com_github_ferranbt_fastssz",
    importpath = "github.com/ferranbt/fastssz",
    sum = "h1:9VDpsWq096+oGMDTT/SgBD/VgZYf4pTF+KTPmZ+OaKM=",
    version = "v0.0.0-20210120143747-11b9eff30ea9",
)

go_repository(
    name = "com_github_flynn_noise",
    importpath = "github.com/flynn/noise",
    sum = "h1:DlTHqmzmvcEiKj+4RYo/imoswx/4r6iBlCMfVtrMXpQ=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_francoispqt_gojay",
    importpath = "github.com/francoispqt/gojay",
    sum = "h1:d2m3sFjloqoIUQU3TsHBgj6qg/BVGlTBeHDUmyJnXKk=",
    version = "v1.2.13",
)

go_repository(
    name = "com_github_fsnotify_fsnotify",
    importpath = "github.com/fsnotify/fsnotify",
    sum = "h1:jRbGcIw6P2Meqdwuo0H1p6JVLbL5DHKAKlYndzMwVZI=",
    version = "v1.5.4",
)

go_repository(
    name = "com_github_ghodss_yaml",
    importpath = "github.com/ghodss/yaml",
    sum = "h1:wQHKEahhL6wmXdzwWG11gIVCkOv05bNOh+Rxn0yngAk=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_go_logr_logr",
    importpath = "github.com/go-logr/logr",
    sum = "h1:2DntVwHkVopvECVRSlL5PSo9eG+cAkDCuckLubN+rq0=",
    version = "v1.2.3",
)

go_repository(
    name = "com_github_go_playground_locales",
    importpath = "github.com/go-playground/locales",
    sum = "h1:u50s323jtVGugKlcYeyzC0etD1HifMjqmJqb8WugfUU=",
    version = "v0.14.0",
)

go_repository(
    name = "com_github_go_playground_universal_translator",
    importpath = "github.com/go-playground/universal-translator",
    sum = "h1:82dyy6p4OuJq4/CByFNOn/jYrnRPArHwAcmLoJZxyho=",
    version = "v0.18.0",
)

go_repository(
    name = "com_github_go_playground_validator_v10",
    importpath = "github.com/go-playground/validator/v10",
    sum = "h1:I7mrTYv78z8k8VXa/qJlOlEXn/nBh+BF8dHX5nt/dr0=",
    version = "v10.10.0",
)

go_repository(
    name = "com_github_go_task_slim_sprig",
    importpath = "github.com/go-task/slim-sprig",
    sum = "h1:p104kn46Q8WdvHunIJ9dAyjPVtrBPhSr3KT2yUst43I=",
    version = "v0.0.0-20210107165309-348f09dbbbc0",
)

go_repository(
    name = "com_github_go_yaml_yaml",
    importpath = "github.com/go-yaml/yaml",
    sum = "h1:RYi2hDdss1u4YE7GwixGzWwVo47T8UQwnTLB6vQiq+o=",
    version = "v2.1.0+incompatible",
)

go_repository(
    name = "com_github_godbus_dbus_v5",
    importpath = "github.com/godbus/dbus/v5",
    sum = "h1:4KLkAxT3aOY8Li4FRJe/KvhoNFFxo0m6fNuFUO8QJUk=",
    version = "v5.1.0",
)

go_repository(
    name = "com_github_gogo_protobuf",
    importpath = "github.com/gogo/protobuf",
    sum = "h1:Ov1cvc58UF3b5XjBnZv7+opcTcQFZebYjWzi34vdm4Q=",
    version = "v1.3.2",
)

go_repository(
    name = "com_github_golang_gddo",
    importpath = "github.com/golang/gddo",
    sum = "h1:HoqgYR60VYu5+0BuG6pjeGp7LKEPZnHt+dUClx9PeIs=",
    version = "v0.0.0-20200528160355-8d077c1d8f4c",
)

go_repository(
    name = "com_github_golang_groupcache",
    importpath = "github.com/golang/groupcache",
    sum = "h1:1r7pUrabqp18hOBcwBwiTsbnFeTZHV9eER/QT5JVZxY=",
    version = "v0.0.0-20200121045136-8c9f03a8e57e",
)

go_repository(
    name = "com_github_golang_mock",
    importpath = "github.com/golang/mock",
    sum = "h1:ErTB+efbowRARo13NNdxyJji2egdxLGQhRaY+DUumQc=",
    version = "v1.6.0",
)

go_repository(
    name = "com_github_google_go_cmp",
    importpath = "github.com/google/go-cmp",
    sum = "h1:O2Tfq5qg4qc4AmwVlvv0oLiVAGB7enBSJ2x2DqQFi38=",
    version = "v0.5.9",
)

go_repository(
    name = "com_github_google_gopacket",
    importpath = "github.com/google/gopacket",
    sum = "h1:ves8RnFZPGiFnTS0uPQStjwru6uO6h+nlr9j6fL7kF8=",
    version = "v1.1.19",
)

go_repository(
    name = "com_github_google_pprof",
    importpath = "github.com/google/pprof",
    sum = "h1:fR20TYVVwhK4O7r7y+McjRYyaTH6/vjwJOajE+XhlzM=",
    version = "v0.0.0-20221203041831-ce31453925ec",
)

go_repository(
    name = "com_github_gorilla_mux",
    importpath = "github.com/gorilla/mux",
    sum = "h1:i40aqfkR1h2SlN9hojwV5ZA91wcXFOvkdNIeFDP5koI=",
    version = "v1.8.0",
)

go_repository(
    name = "com_github_gostaticanalysis_comment",
    importpath = "github.com/gostaticanalysis/comment",
    sum = "h1:hlnx5+S2fY9Zo9ePo4AhgYsYHbM2+eAv8m/s1JiCd6Q=",
    version = "v1.4.2",
)

go_repository(
    name = "com_github_grpc_ecosystem_go_grpc_middleware",
    importpath = "github.com/grpc-ecosystem/go-grpc-middleware",
    sum = "h1:FlFbCRLd5Jr4iYXZufAvgWN6Ao0JrI5chLINnUXDDr0=",
    version = "v1.2.2",
)

go_repository(
    name = "com_github_grpc_ecosystem_go_grpc_prometheus",
    importpath = "github.com/grpc-ecosystem/go-grpc-prometheus",
    sum = "h1:Ovs26xHkKqVztRpIrF/92BcuyuQ/YW4NSIpoGtfXNho=",
    version = "v1.2.0",
)

go_repository(
    name = "com_github_grpc_ecosystem_grpc_gateway_v2",
    importpath = "github.com/grpc-ecosystem/grpc-gateway/v2",
    sum = "h1:X2vfSnm1WC8HEo0MBHZg2TcuDUHJj6kd1TmEAQncnSA=",
    version = "v2.0.1",
)

go_repository(
    name = "com_github_herumi_bls_eth_go_binary",
    importpath = "github.com/herumi/bls-eth-go-binary",
    sum = "h1:wCMygKUQhmcQAjlk2Gquzq6dLmyMv2kF+llRspoRgrk=",
    version = "v0.0.0-20210917013441-d37c07cfda4e",
)

go_repository(
    name = "com_github_holiman_goevmlab",
    importpath = "github.com/holiman/goevmlab",
    sum = "h1:EUpp9uXnEOfAhetF6qy275Lav2Lz0TnJCjTsHl1tf5E=",
    version = "v0.0.0-20220902091028-02faf03e18e4",
)

go_repository(
    name = "com_github_ianlancetaylor_cgosymbolizer",
    importpath = "github.com/ianlancetaylor/cgosymbolizer",
    sum = "h1:IpTHAzWv1pKDDWeJDY5VOHvqc2T9d3C8cPKEf2VPqHE=",
    version = "v0.0.0-20200424224625-be1b05b0b279",
)

go_repository(
    name = "com_github_ipfs_go_cid",
    importpath = "github.com/ipfs/go-cid",
    sum = "h1:OGgOd+JCFM+y1DjWPmVH+2/4POtpDzwcr7VgnB7mZXc=",
    version = "v0.3.2",
)

go_repository(
    name = "com_github_ipfs_go_log",
    importpath = "github.com/ipfs/go-log",
    sum = "h1:2dOuUCB1Z7uoczMWgAyDck5JLb72zHzrMnGnCNNbvY8=",
    version = "v1.0.5",
)

go_repository(
    name = "com_github_ipfs_go_log_v2",
    importpath = "github.com/ipfs/go-log/v2",
    sum = "h1:1XdUzF7048prq4aBjDQQ4SL5RxftpRGdXhNRwKSAlcY=",
    version = "v2.5.1",
)

go_repository(
    name = "com_github_jbenet_go_temp_err_catcher",
    importpath = "github.com/jbenet/go-temp-err-catcher",
    sum = "h1:zpb3ZH6wIE8Shj2sKS+khgRvf7T7RABoLk/+KKHggpk=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_joonix_log",
    importpath = "github.com/joonix/log",
    sum = "h1:k+SfYbN66Ev/GDVq39wYOXVW5RNd5kzzairbCe9dK5Q=",
    version = "v0.0.0-20200409080653-9c1d2ceb5f1d",
)

go_repository(
    name = "com_github_json_iterator_go",
    importpath = "github.com/json-iterator/go",
    sum = "h1:PV8peI4a0ysnczrg+LtxykD8LfKY9ML6u2jnxaEnrnM=",
    version = "v1.1.12",
)

go_repository(
    name = "com_github_juju_ansiterm",
    importpath = "github.com/juju/ansiterm",
    sum = "h1:FaWFmfWdAUKbSCtOU2QjDaorUexogfaMgbipgYATUMU=",
    version = "v0.0.0-20180109212912-720a0952cc2a",
)

go_repository(
    name = "com_github_k0kubun_go_ansi",
    importpath = "github.com/k0kubun/go-ansi",
    sum = "h1:qGQQKEcAR99REcMpsXCp3lJ03zYT1PkRd3kQGPn9GVg=",
    version = "v0.0.0-20180517002512-3bf9e2903213",
)

go_repository(
    name = "com_github_klauspost_compress",
    importpath = "github.com/klauspost/compress",
    sum = "h1:YClS/PImqYbn+UILDnqxQCZ3RehC9N318SU3kElDUEM=",
    version = "v1.15.12",
)

go_repository(
    name = "com_github_klauspost_cpuid_v2",
    importpath = "github.com/klauspost/cpuid/v2",
    sum = "h1:U33DW0aiEj633gHYw3LoDNfkDiYnE5Q8M/TKJn2f2jI=",
    version = "v2.2.1",
)

go_repository(
    name = "com_github_koron_go_ssdp",
    importpath = "github.com/koron/go-ssdp",
    sum = "h1:JivLMY45N76b4p/vsWGOKewBQu6uf39y8l+AQ7sDKx8=",
    version = "v0.0.3",
)

go_repository(
    name = "com_github_kr_pretty",
    importpath = "github.com/kr/pretty",
    sum = "h1:WgNl7dwNpEZ6jJ9k1snq4pZsg7DOEN8hP9Xw0Tsjwk0=",
    version = "v0.3.0",
)

go_repository(
    name = "com_github_kr_text",
    importpath = "github.com/kr/text",
    sum = "h1:5Nx0Ya0ZqY2ygV366QzturHI13Jq95ApcVaJBhpS+AY=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_leodido_go_urn",
    importpath = "github.com/leodido/go-urn",
    sum = "h1:BqpAaACuzVSgi/VLzGZIobT2z4v53pjosyNd9Yv6n/w=",
    version = "v1.2.1",
)

go_repository(
    name = "com_github_libp2p_go_buffer_pool",
    importpath = "github.com/libp2p/go-buffer-pool",
    sum = "h1:oK4mSFcQz7cTQIfqbe4MIj9gLW+mnanjyFtc6cdF0Y8=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_libp2p_go_cidranger",
    importpath = "github.com/libp2p/go-cidranger",
    sum = "h1:ewPN8EZ0dd1LSnrtuwd4709PXVcITVeuwbag38yPW7c=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_libp2p_go_flow_metrics",
    importpath = "github.com/libp2p/go-flow-metrics",
    sum = "h1:0iPhMI8PskQwzh57jB9WxIuIOQ0r+15PChFGkx3Q3WM=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_libp2p_go_libp2p",
    importpath = "github.com/libp2p/go-libp2p",
    sum = "h1:DQk/5bBon+yUVIGTeRVBmOYpZzoBHx/VTC0xoLgJGG4=",
    version = "v0.24.0",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_asn_util",
    importpath = "github.com/libp2p/go-libp2p-asn-util",
    sum = "h1:rg3+Os8jbnO5DxkC7K/Utdi+DkY3q/d1/1q+8WeNAsw=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_libp2p_go_libp2p_pubsub",
    importpath = "github.com/libp2p/go-libp2p-pubsub",
    sum = "h1:KygfDpaa9AeUPGCVcpVenpXNFauDn+5kBYu3EjcL3Tg=",
    version = "v0.8.0",
)

go_repository(
    name = "com_github_libp2p_go_mplex",
    importpath = "github.com/libp2p/go-mplex",
    sum = "h1:BDhFZdlk5tbr0oyFq/xv/NPGfjbnrsDam1EvutpBDbY=",
    version = "v0.7.0",
)

go_repository(
    name = "com_github_libp2p_go_msgio",
    importpath = "github.com/libp2p/go-msgio",
    sum = "h1:W6shmB+FeynDrUVl2dgFQvzfBZcXiyqY4VmpQLu9FqU=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_libp2p_go_nat",
    importpath = "github.com/libp2p/go-nat",
    sum = "h1:MfVsH6DLcpa04Xr+p8hmVRG4juse0s3J8HyNWYHffXg=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_libp2p_go_netroute",
    importpath = "github.com/libp2p/go-netroute",
    sum = "h1:V8kVrpD8GK0Riv15/7VN6RbUQ3URNZVosw7H2v9tksU=",
    version = "v0.2.1",
)

go_repository(
    name = "com_github_libp2p_go_openssl",
    importpath = "github.com/libp2p/go-openssl",
    sum = "h1:LBkKEcUv6vtZIQLVTegAil8jbNpJErQ9AnT+bWV+Ooo=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_libp2p_go_reuseport",
    importpath = "github.com/libp2p/go-reuseport",
    sum = "h1:18PRvIMlpY6ZK85nIAicSBuXXvrYoSw3dsBAR7zc560=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_libp2p_go_yamux_v4",
    importpath = "github.com/libp2p/go-yamux/v4",
    sum = "h1:+Y80dV2Yx/kv7Y7JKu0LECyVdMXm1VUoko+VQ9rBfZQ=",
    version = "v4.0.0",
)

go_repository(
    name = "com_github_logrusorgru_aurora",
    importpath = "github.com/logrusorgru/aurora",
    sum = "h1:tOpm7WcpBTn4fjmVfgpQq0EfczGlG91VSDkswnjF5A8=",
    version = "v2.0.3+incompatible",
)

go_repository(
    name = "com_github_lucas_clemente_quic_go",
    importpath = "github.com/lucas-clemente/quic-go",
    sum = "h1:MfNp3fk0wjWRajw6quMFA3ap1AVtlU+2mtwmbVogB2M=",
    version = "v0.31.0",
)

go_repository(
    name = "com_github_lunixbochs_vtclean",
    importpath = "github.com/lunixbochs/vtclean",
    sum = "h1:xu2sLAri4lGiovBDQKxl5mrXyESr3gUr5m5SM5+LVb8=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_manifoldco_promptui",
    importpath = "github.com/manifoldco/promptui",
    sum = "h1:3l11YT8tm9MnwGFQ4kETwkzpAwY2Jt9lCrumCUW4+z4=",
    version = "v0.7.0",
)

go_repository(
    name = "com_github_mariusvanderwijden_fuzzyvm",
    importpath = "github.com/MariusVanDerWijden/FuzzyVM",
    sum = "h1:ll28mmvFEFWzyXuG/HiFCUMlf2xncN9Yo6c3jCMmA8s=",
    version = "v0.0.0-20220901111237-4348e62e228d",
)

go_repository(
    name = "com_github_mariusvanderwijden_tx_fuzz",
    importpath = "github.com/MariusVanDerWijden/tx-fuzz",
    sum = "h1:sVb99a9AUxf26t06VNnDO5DReatE0lXCp6amDOaalXM=",
    version = "v0.0.0-20220321065247-ebb195301a27",
)

go_repository(
    name = "com_github_marten_seemann_qtls_go1_18",
    importpath = "github.com/marten-seemann/qtls-go1-18",
    sum = "h1:R4H2Ks8P6pAtUagjFty2p7BVHn3XiwDAl7TTQf5h7TI=",
    version = "v0.1.3",
)

go_repository(
    name = "com_github_marten_seemann_qtls_go1_19",
    importpath = "github.com/marten-seemann/qtls-go1-19",
    sum = "h1:mnbxeq3oEyQxQXwI4ReCgW9DPoPR94sNlqWoDZnjRIE=",
    version = "v0.1.1",
)

go_repository(
    name = "com_github_marten_seemann_tcp",
    importpath = "github.com/marten-seemann/tcp",
    sum = "h1:br0buuQ854V8u83wA0rVZ8ttrq5CpaPZdvrK0LP2lOk=",
    version = "v0.0.0-20210406111302-dfbc87cc63fd",
)

go_repository(
    name = "com_github_mattn_go_pointer",
    importpath = "github.com/mattn/go-pointer",
    sum = "h1:n+XhsuGeVO6MEAp7xyEukFINEa+Quek5psIR/ylA6o0=",
    version = "v0.0.1",
)

go_repository(
    name = "com_github_matttproud_golang_protobuf_extensions",
    importpath = "github.com/matttproud/golang_protobuf_extensions",
    sum = "h1:mmDVorXM7PCGKw94cs5zkfA9PSy5pEvNWRP0ET0TIVo=",
    version = "v1.0.4",
)

go_repository(
    name = "com_github_mgutz_ansi",
    importpath = "github.com/mgutz/ansi",
    sum = "h1:j7+1HpAFS1zy5+Q4qx1fWh90gTKwiN4QCGoY9TWyyO4=",
    version = "v0.0.0-20170206155736-9520e82c474b",
)

go_repository(
    name = "com_github_miekg_dns",
    importpath = "github.com/miekg/dns",
    sum = "h1:DQUfb9uc6smULcREF09Uc+/Gd46YWqJd5DbpPE9xkcA=",
    version = "v1.1.50",
)

go_repository(
    name = "com_github_mikioh_tcpinfo",
    importpath = "github.com/mikioh/tcpinfo",
    sum = "h1:z78hV3sbSMAUoyUMM0I83AUIT6Hu17AWfgjzIbtrYFc=",
    version = "v0.0.0-20190314235526-30a79bb1804b",
)

go_repository(
    name = "com_github_mikioh_tcpopt",
    importpath = "github.com/mikioh/tcpopt",
    sum = "h1:PTfri+PuQmWDqERdnNMiD9ZejrlswWrCpBEZgWOiTrc=",
    version = "v0.0.0-20190314235656-172688c1accc",
)

go_repository(
    name = "com_github_minio_highwayhash",
    importpath = "github.com/minio/highwayhash",
    sum = "h1:dZ6IIu8Z14VlC0VpfKofAhCy74wu/Qb5gcn52yWoz/0=",
    version = "v1.0.1",
)

go_repository(
    name = "com_github_minio_sha256_simd",
    importpath = "github.com/minio/sha256-simd",
    sum = "h1:v1ta+49hkWZyvaKwrQB8elexRqm6Y0aMLjCNsrYxo6g=",
    version = "v1.0.0",
)

go_repository(
    name = "com_github_mitchellh_colorstring",
    importpath = "github.com/mitchellh/colorstring",
    sum = "h1:62I3jR2EmQ4l5rM/4FEfDWcRD+abF5XlKShorW5LRoQ=",
    version = "v0.0.0-20190213212951-d06e56a500db",
)

go_repository(
    name = "com_github_modern_go_concurrent",
    importpath = "github.com/modern-go/concurrent",
    sum = "h1:TRLaZ9cD/w8PVh93nsPXa1VrQ6jlwL5oN8l14QlcNfg=",
    version = "v0.0.0-20180306012644-bacd9c7ef1dd",
)

go_repository(
    name = "com_github_modern_go_reflect2",
    importpath = "github.com/modern-go/reflect2",
    sum = "h1:xBagoLtFs94CBntxluKeaWgTMpvLxC4ur3nMaC9Gz0M=",
    version = "v1.0.2",
)

go_repository(
    name = "com_github_mohae_deepcopy",
    importpath = "github.com/mohae/deepcopy",
    sum = "h1:RWengNIwukTxcDr9M+97sNutRR1RKhG96O6jWumTTnw=",
    version = "v0.0.0-20170929034955-c48cc78d4826",
)

go_repository(
    name = "com_github_mr_tron_base58",
    importpath = "github.com/mr-tron/base58",
    sum = "h1:T/HDJBh4ZCPbU39/+c3rRvE0uKBQlU27+QI8LJ4t64o=",
    version = "v1.2.0",
)

go_repository(
    name = "com_github_multiformats_go_base32",
    importpath = "github.com/multiformats/go-base32",
    sum = "h1:pVx9xoSPqEIQG8o+UbAe7DNi51oej1NtK+aGkbLYxPE=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_multiformats_go_base36",
    importpath = "github.com/multiformats/go-base36",
    sum = "h1:lFsAbNOGeKtuKozrtBsAkSVhv1p9D0/qedU9rQyccr0=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_multiformats_go_multiaddr",
    importpath = "github.com/multiformats/go-multiaddr",
    sum = "h1:aqjksEcqK+iD/Foe1RRFsGZh8+XFiGo7FgUCZlpv3LU=",
    version = "v0.8.0",
)

go_repository(
    name = "com_github_multiformats_go_multiaddr_dns",
    importpath = "github.com/multiformats/go-multiaddr-dns",
    sum = "h1:QgQgR+LQVt3NPTjbrLLpsaT2ufAA2y0Mkk+QRVJbW3A=",
    version = "v0.3.1",
)

go_repository(
    name = "com_github_multiformats_go_multiaddr_fmt",
    importpath = "github.com/multiformats/go-multiaddr-fmt",
    sum = "h1:WLEFClPycPkp4fnIzoFoV9FVd49/eQsuaL3/CWe167E=",
    version = "v0.1.0",
)

go_repository(
    name = "com_github_multiformats_go_multibase",
    importpath = "github.com/multiformats/go-multibase",
    sum = "h1:3ASCDsuLX8+j4kx58qnJ4YFq/JWTJpCyDW27ztsVTOI=",
    version = "v0.1.1",
)

go_repository(
    name = "com_github_multiformats_go_multicodec",
    importpath = "github.com/multiformats/go-multicodec",
    sum = "h1:rTUjGOwjlhGHbEMbPoSUJowG1spZTVsITRANCjKTUAQ=",
    version = "v0.7.0",
)

go_repository(
    name = "com_github_multiformats_go_multihash",
    importpath = "github.com/multiformats/go-multihash",
    sum = "h1:aem8ZT0VA2nCHHk7bPJ1BjUbHNciqZC/d16Vve9l108=",
    version = "v0.2.1",
)

go_repository(
    name = "com_github_multiformats_go_multistream",
    importpath = "github.com/multiformats/go-multistream",
    sum = "h1:d5PZpjwRgVlbwfdTDjife7XszfZd8KYWfROYFlGcR8o=",
    version = "v0.3.3",
)

go_repository(
    name = "com_github_multiformats_go_varint",
    importpath = "github.com/multiformats/go-varint",
    sum = "h1:sWSGR+f/eu5ABZA2ZpYKBILXTTs9JWpdEM/nEGOHFS8=",
    version = "v0.0.7",
)

go_repository(
    name = "com_github_nxadm_tail",
    importpath = "github.com/nxadm/tail",
    sum = "h1:nPr65rt6Y5JFSKQO7qToXr7pePgD6Gwiw05lkbyAQTE=",
    version = "v1.4.8",
)

go_repository(
    name = "com_github_onsi_ginkgo",
    importpath = "github.com/onsi/ginkgo",
    sum = "h1:8xi0RTUf59SOSfEtZMvwTvXYMzG4gV23XVHOZiXNtnE=",
    version = "v1.16.5",
)

go_repository(
    name = "com_github_onsi_ginkgo_v2",
    importpath = "github.com/onsi/ginkgo/v2",
    sum = "h1:auzK7OI497k6x4OvWq+TKAcpcSAlod0doAH72oIN0Jw=",
    version = "v2.5.1",
)

go_repository(
    name = "com_github_onsi_gomega",
    importpath = "github.com/onsi/gomega",
    sum = "h1:+0glovB9Jd6z3VR+ScSwQqXVTIfJcGA9UBM8yzQxhqg=",
    version = "v1.24.0",
)

go_repository(
    name = "com_github_opencontainers_runtime_spec",
    importpath = "github.com/opencontainers/runtime-spec",
    sum = "h1:UfAcuLBJB9Coz72x1hgl8O5RVzTdNiaglX6v2DM6FI0=",
    version = "v1.0.2",
)

go_repository(
    name = "com_github_patrickmn_go_cache",
    importpath = "github.com/patrickmn/go-cache",
    sum = "h1:HRMgzkcYKYpi3C8ajMPV8OFXaaRUnok+kx1WdO15EQc=",
    version = "v2.1.0+incompatible",
)

go_repository(
    name = "com_github_paulbellamy_ratecounter",
    importpath = "github.com/paulbellamy/ratecounter",
    sum = "h1:2L/RhJq+HA8gBQImDXtLPrDXK5qAj6ozWVK/zFXVJGs=",
    version = "v0.2.0",
)

go_repository(
    name = "com_github_pbnjay_memory",
    importpath = "github.com/pbnjay/memory",
    sum = "h1:onHthvaw9LFnH4t2DcNVpwGmV9E1BkGknEliJkfwQj0=",
    version = "v0.0.0-20210728143218-7b4eea64cf58",
)

go_repository(
    name = "com_github_pborman_uuid",
    importpath = "github.com/pborman/uuid",
    sum = "h1:+ZZIw58t/ozdjRaXh/3awHfmWRbzYxJoAdNJxe/3pvw=",
    version = "v1.2.1",
)

go_repository(
    name = "com_github_prometheus_client_golang",
    importpath = "github.com/prometheus/client_golang",
    sum = "h1:nJdhIvne2eSX/XRAFV9PcvFFRbrjbcTUj0VP62TMhnw=",
    version = "v1.14.0",
)

go_repository(
    name = "com_github_prometheus_client_model",
    importpath = "github.com/prometheus/client_model",
    sum = "h1:UBgGFHqYdG/TPFD1B1ogZywDqEkwp3fBMvqdiQ7Xew4=",
    version = "v0.3.0",
)

go_repository(
    name = "com_github_prometheus_common",
    importpath = "github.com/prometheus/common",
    sum = "h1:ccBbHCgIiT9uSoFY0vX8H3zsNR5eLt17/RQLUvn8pXE=",
    version = "v0.37.0",
)

go_repository(
    name = "com_github_prometheus_procfs",
    importpath = "github.com/prometheus/procfs",
    sum = "h1:ODq8ZFEaYeCaZOJlZZdJA2AbQR98dSHSM1KW/You5mo=",
    version = "v0.8.0",
)

go_repository(
    name = "com_github_prometheus_prom2json",
    importpath = "github.com/prometheus/prom2json",
    sum = "h1:BlqrtbT9lLH3ZsOVhXPsHzFrApCTKRifB7gjJuypu6Y=",
    version = "v1.3.0",
)

go_repository(
    name = "com_github_prysmaticlabs_fastssz",
    importpath = "github.com/prysmaticlabs/fastssz",
    sum = "h1:Y3PcvUrnneMWLuypZpwPz8P70/DQsz6KgV9JveKpyZs=",
    version = "v0.0.0-20220628121656-93dfe28febab",
)

go_repository(
    name = "com_github_prysmaticlabs_go_bitfield",
    importpath = "github.com/prysmaticlabs/go-bitfield",
    sum = "h1:0tVE4tdWQK9ZpYygoV7+vS6QkDvQVySboMVEIxBJmXw=",
    version = "v0.0.0-20210809151128-385d8c5e3fb7",
)

go_repository(
    name = "com_github_prysmaticlabs_gohashtree",
    importpath = "github.com/prysmaticlabs/gohashtree",
    sum = "h1:hk5ZsDQuSkyUMhTd55qB396P1+dtyIKiSwMmYE/hyEU=",
    version = "v0.0.2-alpha",
)

go_repository(
    name = "com_github_prysmaticlabs_prombbolt",
    importpath = "github.com/prysmaticlabs/prombbolt",
    sum = "h1:9PHRCuO/VN0s9k+RmLykho7AjDxblNYI5bYKed16NPU=",
    version = "v0.0.0-20210126082820-9b7adba6db7c",
)

go_repository(
    name = "com_github_prysmaticlabs_protoc_gen_go_cast",
    importpath = "github.com/prysmaticlabs/protoc-gen-go-cast",
    sum = "h1:+jhXLjEYVW4qU2z5SOxlxN+Hv/A9FDf0HpfDurfMEz0=",
    version = "v0.0.0-20211014160335-757fae4f38c6",
)

go_repository(
    name = "com_github_prysmaticlabs_prysm_v3",
    importpath = "github.com/prysmaticlabs/prysm/v3",
    sum = "h1:r+a+3x2yBg8mPba+2GZRzHjSdp7WZnXm5HZnq/ka6WE=",
    version = "v3.2.0",
)

go_repository(
    name = "com_github_r3labs_sse",
    importpath = "github.com/r3labs/sse",
    sum = "h1:zAsgcP8MhzAbhMnB1QQ2O7ZhWYVGYSR2iVcjzQuPV+o=",
    version = "v0.0.0-20210224172625-26fe804710bc",
)

go_repository(
    name = "com_github_raulk_go_watchdog",
    importpath = "github.com/raulk/go-watchdog",
    sum = "h1:oUmdlHxdkXRJlwfG0O9omj8ukerm8MEQavSiDTEtBsk=",
    version = "v1.3.0",
)

go_repository(
    name = "com_github_rivo_uniseg",
    importpath = "github.com/rivo/uniseg",
    sum = "h1:3Z3Eu6FGHZWSfNKJTOUiPatWwfc7DzJRU04jFUqJODw=",
    version = "v0.3.4",
)

go_repository(
    name = "com_github_rogpeppe_go_internal",
    importpath = "github.com/rogpeppe/go-internal",
    sum = "h1:FCbCCtXNOY3UtUuHUYaghJg4y7Fd14rXifAYUAtL9R8=",
    version = "v1.8.0",
)

go_repository(
    name = "com_github_schollz_progressbar_v3",
    importpath = "github.com/schollz/progressbar/v3",
    sum = "h1:nMinx+JaEm/zJz4cEyClQeAw5rsYSB5th3xv+5lV6Vg=",
    version = "v3.3.4",
)

go_repository(
    name = "com_github_spacemonkeygo_spacelog",
    importpath = "github.com/spacemonkeygo/spacelog",
    sum = "h1:RC6RW7j+1+HkWaX/Yh71Ee5ZHaHYt7ZP4sQgUrm6cDU=",
    version = "v0.0.0-20180420211403-2296661a0572",
)

go_repository(
    name = "com_github_spaolacci_murmur3",
    importpath = "github.com/spaolacci/murmur3",
    sum = "h1:7c1g84S4BPRrfL5Xrdp6fOJ206sU9y293DDHaoy0bLI=",
    version = "v1.1.0",
)

go_repository(
    name = "com_github_thomaso_mirodin_intmath",
    importpath = "github.com/thomaso-mirodin/intmath",
    sum = "h1:cR8/SYRgyQCt5cNCMniB/ZScMkhI9nk8U5C7SbISXjo=",
    version = "v0.0.0-20160323211736-5dc6d854e46e",
)

go_repository(
    name = "com_github_trailofbits_go_mutexasserts",
    importpath = "github.com/trailofbits/go-mutexasserts",
    sum = "h1:8LRP+2JK8piIUU16ZDgWDXwjJcuJNTtCzadjTZj8Jf0=",
    version = "v0.0.0-20200708152505-19999e7d3cef",
)

go_repository(
    name = "com_github_uber_jaeger_client_go",
    importpath = "github.com/uber/jaeger-client-go",
    sum = "h1:IxcNZ7WRY1Y3G4poYlx24szfsn/3LvK9QHCq9oQw8+U=",
    version = "v2.25.0+incompatible",
)

go_repository(
    name = "com_github_uudashr_gocognit",
    importpath = "github.com/uudashr/gocognit",
    sum = "h1:rrSex7oHr3/pPLQ0xoWq108XMU8s678FJcQ+aSfOHa4=",
    version = "v1.0.5",
)

go_repository(
    name = "com_github_wealdtech_go_bytesutil",
    importpath = "github.com/wealdtech/go-bytesutil",
    sum = "h1:ocEg3Ke2GkZ4vQw5lp46rmO+pfqCCTgq35gqOy8JKVc=",
    version = "v1.1.1",
)

go_repository(
    name = "com_github_wealdtech_go_eth2_types_v2",
    importpath = "github.com/wealdtech/go-eth2-types/v2",
    sum = "h1:tiA6T88M6XQIbrV5Zz53l1G5HtRERcxQfmET225V4Ls=",
    version = "v2.5.2",
)

go_repository(
    name = "com_github_wealdtech_go_eth2_util",
    importpath = "github.com/wealdtech/go-eth2-util",
    sum = "h1:2INPeOR35x5LdFFpSzyw954WzTD+DFyHe3yKlJnG5As=",
    version = "v1.6.3",
)

go_repository(
    name = "com_github_wealdtech_go_eth2_wallet_encryptor_keystorev4",
    importpath = "github.com/wealdtech/go-eth2-wallet-encryptor-keystorev4",
    sum = "h1:SxrDVSr+oXuT1x8kZt4uWqNCvv5xXEGV9zd7cuSrZS8=",
    version = "v1.1.3",
)

go_repository(
    name = "com_github_wercker_journalhook",
    importpath = "github.com/wercker/journalhook",
    sum = "h1:shC1HB1UogxN5Ech3Yqaaxj1X/P656PPCB4RbojIJqc=",
    version = "v0.0.0-20180428041537-5d0a5ae867b3",
)

go_repository(
    name = "com_github_whyrusleeping_timecache",
    importpath = "github.com/whyrusleeping/timecache",
    sum = "h1:lYbXeSvJi5zk5GLKVuid9TVjS9a0OmLIDKTfoZBL6Ow=",
    version = "v0.0.0-20160911033111-cfcb2f1abfee",
)

go_repository(
    name = "com_github_yusufpapurcu_wmi",
    importpath = "github.com/yusufpapurcu/wmi",
    sum = "h1:KBNDSne4vP5mbSWnJbO+51IMOXJB67QiYCSBrubbPRg=",
    version = "v1.2.2",
)

go_repository(
    name = "com_lukechampine_blake3",
    importpath = "lukechampine.com/blake3",
    sum = "h1:GgRMhmdsuK8+ii6UZFDL8Nb+VyMwadAgcJyfYHxG6n0=",
    version = "v1.1.7",
)

go_repository(
    name = "in_gopkg_cenkalti_backoff_v1",
    importpath = "gopkg.in/cenkalti/backoff.v1",
    sum = "h1:Arh75ttbsvlpVA7WtVpH4u9h6Zl46xuptxqLxPiSo4Y=",
    version = "v1.1.0",
)

go_repository(
    name = "in_gopkg_d4l3k_messagediff_v1",
    importpath = "gopkg.in/d4l3k/messagediff.v1",
    sum = "h1:70AthpjunwzUiarMHyED52mj9UwtAnE89l1Gmrt3EU0=",
    version = "v1.2.1",
)

go_repository(
    name = "in_gopkg_inf_v0",
    importpath = "gopkg.in/inf.v0",
    sum = "h1:73M5CoZyi3ZLMOyDlQh031Cx6N9NDJ2Vvfl76EDAgDc=",
    version = "v0.9.1",
)

go_repository(
    name = "in_gopkg_tomb_v1",
    importpath = "gopkg.in/tomb.v1",
    sum = "h1:uRGJdciOHaEIrze2W8Q3AKkepLTh2hOroT7a+7czfdQ=",
    version = "v1.0.0-20141024135613-dd632973f1e7",
)

go_repository(
    name = "io_etcd_go_bbolt",
    importpath = "go.etcd.io/bbolt",
    sum = "h1:XAzx9gjCb0Rxj7EoqcClPD1d5ZBxZJk0jbuoPHenBt0=",
    version = "v1.3.5",
)

go_repository(
    name = "io_k8s_apimachinery",
    importpath = "k8s.io/apimachinery",
    sum = "h1:pOGcbVAhxADgUYnjS08EFXs9QMl8qaH5U4fr5LGUrSk=",
    version = "v0.18.3",
)

go_repository(
    name = "io_k8s_client_go",
    importpath = "k8s.io/client-go",
    sum = "h1:QaJzz92tsN67oorwzmoB0a9r9ZVHuD5ryjbCKP0U22k=",
    version = "v0.18.3",
)

go_repository(
    name = "io_k8s_klog",
    importpath = "k8s.io/klog",
    sum = "h1:Pt+yjF5aB1xDSVbau4VsWe+dQNzA0qv1LlXdC2dF6Q8=",
    version = "v1.0.0",
)

go_repository(
    name = "io_k8s_klog_v2",
    importpath = "k8s.io/klog/v2",
    sum = "h1:lyJt0TWMPaGoODa8B8bUuxgHS3W/m/bNr2cca3brA/g=",
    version = "v2.80.0",
)

go_repository(
    name = "io_k8s_sigs_structured_merge_diff_v3",
    importpath = "sigs.k8s.io/structured-merge-diff/v3",
    sum = "h1:dOmIZBMfhcHS09XZkMyUgkq5trg3/jRyJYFZUiaOp8E=",
    version = "v3.0.0",
)

go_repository(
    name = "io_k8s_sigs_yaml",
    importpath = "sigs.k8s.io/yaml",
    sum = "h1:kr/MCeFWJWTwyaHoR9c8EjH9OumOmoF9YGiZd7lFm/Q=",
    version = "v1.2.0",
)

go_repository(
    name = "io_k8s_utils",
    importpath = "k8s.io/utils",
    sum = "h1:ZtTUW5+ZWaoqjR3zOpRa7oFJ5d4aA22l4me/xArfOIc=",
    version = "v0.0.0-20200520001619-278ece378a50",
)

go_repository(
    name = "io_opencensus_go",
    importpath = "go.opencensus.io",
    sum = "h1:y73uSU6J157QMP2kn2r30vwW1A2W2WFwSCGnAVxeaD0=",
    version = "v0.24.0",
)

go_repository(
    name = "io_opencensus_go_contrib_exporter_jaeger",
    importpath = "contrib.go.opencensus.io/exporter/jaeger",
    sum = "h1:yGBYzYMewVL0yO9qqJv3Z5+IRhPdU7e9o/2oKpX4YvI=",
    version = "v0.2.1",
)

go_repository(
    name = "org_golang_google_api",
    importpath = "google.golang.org/api",
    sum = "h1:k40adF3uR+6x/+hO5Dh4ZFUqFp67vxvbpafFiJxl10A=",
    version = "v0.34.0",
)

go_repository(
    name = "org_golang_google_appengine",
    importpath = "google.golang.org/appengine",
    sum = "h1:FZR1q0exgwxzPzp/aF+VccGrSfxfPpkBqjIIEq3ru6c=",
    version = "v1.6.7",
)

go_repository(
    name = "org_golang_google_genproto",
    importpath = "google.golang.org/genproto",
    sum = "h1:KMgpo2lWy1vfrYjtxPAzR0aNWeAR1UdQykt6sj/hpBY=",
    version = "v0.0.0-20210426193834-eac7f76ac494",
)

go_repository(
    name = "org_golang_google_grpc",
    importpath = "google.golang.org/grpc",
    sum = "h1:AGJ0Ih4mHjSeibYkFGh1dD9KJ/eOtZ93I6hoHhukQ5Q=",
    version = "v1.40.0",
)

go_repository(
    name = "org_golang_x_oauth2",
    importpath = "golang.org/x/oauth2",
    sum = "h1:clP8eMhB30EHdc0bd2Twtq6kgU7yl5ub2cQLSdrv1Dg=",
    version = "v0.0.0-20220223155221-ee480838109b",
)

go_repository(
    name = "org_uber_go_atomic",
    importpath = "go.uber.org/atomic",
    sum = "h1:9qC72Qh0+3MqyJbAn8YU5xVq1frD8bn3JtD2oXtafVQ=",
    version = "v1.10.0",
)

go_repository(
    name = "org_uber_go_automaxprocs",
    importpath = "go.uber.org/automaxprocs",
    sum = "h1:II28aZoGdaglS5vVNnspf28lnZpXScxtIozx1lAjdb0=",
    version = "v1.3.0",
)

go_repository(
    name = "org_uber_go_dig",
    importpath = "go.uber.org/dig",
    sum = "h1:vq3YWr8zRj1eFGC7Gvf907hE0eRjPTZ1d3xHadD6liE=",
    version = "v1.15.0",
)

go_repository(
    name = "org_uber_go_fx",
    importpath = "go.uber.org/fx",
    sum = "h1:bUNI6oShr+OVFQeU8cDNbnN7VFsu+SsjHzUF51V/GAU=",
    version = "v1.18.2",
)

go_repository(
    name = "org_uber_go_multierr",
    importpath = "go.uber.org/multierr",
    sum = "h1:dg6GjLku4EH+249NNmoIciG9N/jURbDG+pFlTkhzIC8=",
    version = "v1.8.0",
)

go_repository(
    name = "org_uber_go_zap",
    importpath = "go.uber.org/zap",
    sum = "h1:FiJd5l1UOLj0wCgbSE0rwwXHzEdAZS6hiiSnxJN/D60=",
    version = "v1.24.0",
)

# gazelle:repository_macro deps.bzl%go_dependencies
go_dependencies()

go_rules_dependencies()

go_register_toolchains(
    nogo = "@//:nogo",
    version = "1.19.3",
)

gazelle_dependencies()
