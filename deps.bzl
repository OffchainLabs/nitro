load("@bazel_gazelle//:deps.bzl", "go_repository")

def go_dependencies():
    go_repository(
        name = "com_github_alecthomas_template",
        importpath = "github.com/alecthomas/template",
        sum = "h1:cAKDfWh5VpdgMhJosfJnn5/FoN2SRZ4p7fJNX58YPaU=",
        version = "v0.0.0-20160405071501-a0175ee3bccc",
    )

    go_repository(
        name = "com_github_alecthomas_units",
        importpath = "github.com/alecthomas/units",
        sum = "h1:E/8AP5dFtMhl5KPJz66Kt9G0n+7Sn41Fy1wv9/jHOrc=",
        version = "v0.0.0-20210927113745-59d0afb8317a",
    )
    go_repository(
        name = "com_github_alexbrainman_goissue34681",
        importpath = "github.com/alexbrainman/goissue34681",
        sum = "h1:iW0a5ljuFxkLGPNem5Ui+KBjFJzKg4Fv2fnxe4dvzpM=",
        version = "v0.0.0-20191006012335-3fc7a47baff5",
    )
    go_repository(
        name = "com_github_alicebob_gopher_json",
        importpath = "github.com/alicebob/gopher-json",
        sum = "h1:HbKu58rmZpUGpz5+4FfNmIU+FmZg2P3Xaj2v2bfNWmk=",
        version = "v0.0.0-20200520072559-a9ecdc9d1d3a",
    )
    go_repository(
        name = "com_github_alicebob_miniredis_v2",
        importpath = "github.com/alicebob/miniredis/v2",
        sum = "h1:CdmwIlKUWFBDS+4464GtQiQ0R1vpzOgu4Vnd74rBL7M=",
        version = "v2.21.0",
    )

    go_repository(
        name = "com_github_allegro_bigcache",
        importpath = "github.com/allegro/bigcache",
        sum = "h1:eMwmnE/GDgah4HI848JfFxHt+iPb26b4zyfspmqY0/8=",
        version = "v1.2.1-0.20190218064605-e24eb225f156",
    )
    go_repository(
        name = "com_github_andreasbriese_bbloom",
        importpath = "github.com/AndreasBriese/bbloom",
        sum = "h1:cTp8I5+VIoKjsnZuH8vjyaysT/ses3EvZeaV/1UkF2M=",
        version = "v0.0.0-20190825152654-46b345b51c96",
    )
    go_repository(
        name = "com_github_andybalholm_brotli",
        importpath = "github.com/andybalholm/brotli",
        sum = "h1:fpcw+r1N1h0Poc1F/pHbW40cUm/lMEQslZtCkBQ0UnM=",
        version = "v1.0.3",
    )

    go_repository(
        name = "com_github_aws_aws_sdk_go_v2",
        importpath = "github.com/aws/aws-sdk-go-v2",
        sum = "h1:swQTEQUyJF/UkEA94/Ga55miiKFoXmm/Zd67XHgmjSg=",
        version = "v1.16.4",
    )
    go_repository(
        name = "com_github_aws_aws_sdk_go_v2_aws_protocol_eventstream",
        importpath = "github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream",
        sum = "h1:SdK4Ppk5IzLs64ZMvr6MrSficMtjY2oS0WOORXTlxwU=",
        version = "v1.4.1",
    )

    go_repository(
        name = "com_github_aws_aws_sdk_go_v2_config",
        importpath = "github.com/aws/aws-sdk-go-v2/config",
        sum = "h1:P+xwhr6kabhxDTXTVH9YoHkqjLJ0wVVpIUHtFNr2hjU=",
        version = "v1.15.5",
    )
    go_repository(
        name = "com_github_aws_aws_sdk_go_v2_credentials",
        importpath = "github.com/aws/aws-sdk-go-v2/credentials",
        sum = "h1:4R/NqlcRFSkR0wxOhgHi+agGpbEr5qMCjn7VqUIJY+E=",
        version = "v1.12.0",
    )
    go_repository(
        name = "com_github_aws_aws_sdk_go_v2_feature_ec2_imds",
        importpath = "github.com/aws/aws-sdk-go-v2/feature/ec2/imds",
        sum = "h1:FP8gquGeGHHdfY6G5llaMQDF+HAf20VKc8opRwmjf04=",
        version = "v1.12.4",
    )
    go_repository(
        name = "com_github_aws_aws_sdk_go_v2_feature_s3_manager",
        importpath = "github.com/aws/aws-sdk-go-v2/feature/s3/manager",
        sum = "h1:JL7cY85hyjlgfA29MMyAlItX+JYIH9XsxgMBS7jtlqA=",
        version = "v1.11.10",
    )
    go_repository(
        name = "com_github_aws_aws_sdk_go_v2_internal_configsources",
        importpath = "github.com/aws/aws-sdk-go-v2/internal/configsources",
        sum = "h1:gsqHplNh1DaQunEKZISK56wlpbCg0yKxNVvGWCFuF1k=",
        version = "v1.1.11",
    )
    go_repository(
        name = "com_github_aws_aws_sdk_go_v2_internal_endpoints_v2",
        importpath = "github.com/aws/aws-sdk-go-v2/internal/endpoints/v2",
        sum = "h1:PLFj+M2PgIDHG//hw3T0O0KLI4itVtAjtxrZx4AHPLg=",
        version = "v2.4.5",
    )
    go_repository(
        name = "com_github_aws_aws_sdk_go_v2_internal_ini",
        importpath = "github.com/aws/aws-sdk-go-v2/internal/ini",
        sum = "h1:6cZRymlLEIlDTEB0+5+An6Zj1CKt6rSE69tOmFeu1nk=",
        version = "v1.3.11",
    )
    go_repository(
        name = "com_github_aws_aws_sdk_go_v2_internal_v4a",
        importpath = "github.com/aws/aws-sdk-go-v2/internal/v4a",
        sum = "h1:C21IDZCm9Yu5xqjb3fKmxDoYvJXtw1DNlOmLZEIlY1M=",
        version = "v1.0.1",
    )
    go_repository(
        name = "com_github_aws_aws_sdk_go_v2_service_internal_accept_encoding",
        importpath = "github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding",
        sum = "h1:T4pFel53bkHjL2mMo+4DKE6r6AuoZnM0fg7k1/ratr4=",
        version = "v1.9.1",
    )
    go_repository(
        name = "com_github_aws_aws_sdk_go_v2_service_internal_checksum",
        importpath = "github.com/aws/aws-sdk-go-v2/service/internal/checksum",
        sum = "h1:9LSZqt4v1JiehyZTrQnRFf2mY/awmyYNNY/b7zqtduU=",
        version = "v1.1.5",
    )

    go_repository(
        name = "com_github_aws_aws_sdk_go_v2_service_internal_presigned_url",
        importpath = "github.com/aws/aws-sdk-go-v2/service/internal/presigned-url",
        sum = "h1:b16QW0XWl0jWjLABFc1A+uh145Oqv+xDcObNk0iQgUk=",
        version = "v1.9.4",
    )
    go_repository(
        name = "com_github_aws_aws_sdk_go_v2_service_internal_s3shared",
        importpath = "github.com/aws/aws-sdk-go-v2/service/internal/s3shared",
        sum = "h1:RE/DlZLYrz1OOmq8F28IXHLksuuvlpzUbvJ+SESCZBI=",
        version = "v1.13.4",
    )
    go_repository(
        name = "com_github_aws_aws_sdk_go_v2_service_route53",
        importpath = "github.com/aws/aws-sdk-go-v2/service/route53",
        sum = "h1:cKr6St+CtC3/dl/rEBJvlk7A/IN5D5F02GNkGzfbtVU=",
        version = "v1.1.1",
    )

    go_repository(
        name = "com_github_aws_aws_sdk_go_v2_service_s3",
        importpath = "github.com/aws/aws-sdk-go-v2/service/s3",
        sum = "h1:LCQKnopq2t4oQS3VKivlYTzAHCTJZZoQICM9fny7KHY=",
        version = "v1.26.9",
    )

    go_repository(
        name = "com_github_aws_aws_sdk_go_v2_service_sso",
        importpath = "github.com/aws/aws-sdk-go-v2/service/sso",
        sum = "h1:Uw5wBybFQ1UeA9ts0Y07gbv0ncZnIAyw858tDW0NP2o=",
        version = "v1.11.4",
    )
    go_repository(
        name = "com_github_aws_aws_sdk_go_v2_service_sts",
        importpath = "github.com/aws/aws-sdk-go-v2/service/sts",
        sum = "h1:+xtV90n3abQmgzk1pS++FdxZTrPEDgQng6e4/56WR2A=",
        version = "v1.16.4",
    )
    go_repository(
        name = "com_github_aws_smithy_go",
        importpath = "github.com/aws/smithy-go",
        sum = "h1:eG/N+CcUMAvsdffgMvjMKwfyDzIkjM6pfxMJ8Mzc6mE=",
        version = "v1.11.2",
    )
    go_repository(
        name = "com_github_azure_azure_sdk_for_go_sdk_azcore",
        importpath = "github.com/Azure/azure-sdk-for-go/sdk/azcore",
        sum = "h1:qoVeMsc9/fh/yhxVaA0obYjVH/oI/ihrOoMwsLS9KSA=",
        version = "v0.21.1",
    )
    go_repository(
        name = "com_github_azure_azure_sdk_for_go_sdk_internal",
        importpath = "github.com/Azure/azure-sdk-for-go/sdk/internal",
        sum = "h1:E+m3SkZCN0Bf5q7YdTs5lSm2CYY3CK4spn5OmUIiQtk=",
        version = "v0.8.3",
    )
    go_repository(
        name = "com_github_azure_azure_sdk_for_go_sdk_storage_azblob",
        importpath = "github.com/Azure/azure-sdk-for-go/sdk/storage/azblob",
        sum = "h1:Px2UA+2RvSSvv+RvJNuUB6n7rs5Wsel4dXLe90Um2n4=",
        version = "v0.3.0",
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
        name = "com_github_blang_semver_v4",
        importpath = "github.com/blang/semver/v4",
        sum = "h1:1PFHFE6yCCTv8C1TeyNNarDzntLi7wMI5i/pzqYIsAM=",
        version = "v4.0.0",
    )

    go_repository(
        name = "com_github_btcsuite_btcd_btcec_v2",
        importpath = "github.com/btcsuite/btcd/btcec/v2",
        sum = "h1:5n0X6hX0Zk+6omWcihdYvdAlGf2DfasC0GMf7DClJ3U=",
        version = "v2.3.2",
    )
    go_repository(
        name = "com_github_btcsuite_btcd_chaincfg_chainhash",
        importpath = "github.com/btcsuite/btcd/chaincfg/chainhash",
        sum = "h1:q0rUy8C/TYNBQS1+CGKw68tLOFYSNEs0TFnxxnS9+4U=",
        version = "v1.0.1",
    )

    go_repository(
        name = "com_github_cavaliergopher_grab_v3",
        importpath = "github.com/cavaliergopher/grab/v3",
        sum = "h1:4z7TkBfmPjmLAAmkkAZNX/6QJ1nNFdv3SdIHXju0Fr4=",
        version = "v3.0.1",
    )
    go_repository(
        name = "com_github_cenkalti_backoff",
        importpath = "github.com/cenkalti/backoff",
        sum = "h1:tNowT99t7UNflLxfYYSlKYsBpXdEet03Pg2g16Swow4=",
        version = "v2.2.1+incompatible",
    )
    go_repository(
        name = "com_github_cenkalti_backoff_v4",
        importpath = "github.com/cenkalti/backoff/v4",
        sum = "h1:cFAlzYUlVYDysBEH2T5hyJZMh3+5+WCBvSnK6Q8UtC4=",
        version = "v4.1.3",
    )

    go_repository(
        name = "com_github_ceramicnetwork_go_dag_jose",
        importpath = "github.com/ceramicnetwork/go-dag-jose",
        sum = "h1:yJ/HVlfKpnD3LdYP03AHyTvbm3BpPiz2oZiOeReJRdU=",
        version = "v0.1.0",
    )
    go_repository(
        name = "com_github_cespare_cp",
        importpath = "github.com/cespare/cp",
        sum = "h1:SE+dxFebS7Iik5LK0tsi1k9ZCxEaFX4AjQmoyA+1dJk=",
        version = "v0.1.0",
    )

    go_repository(
        name = "com_github_cespare_xxhash",
        importpath = "github.com/cespare/xxhash",
        sum = "h1:a6HrQnmkObjyL+Gs60czilIUGqrzKutQD6XZog3p+ko=",
        version = "v1.1.0",
    )

    go_repository(
        name = "com_github_cespare_xxhash_v2",
        importpath = "github.com/cespare/xxhash/v2",
        sum = "h1:DC2CZ1Ep5Y4k3ZQ899DldepgrayRUGE6BBZ/cd9Cj44=",
        version = "v2.2.0",
    )
    go_repository(
        name = "com_github_cloudflare_cloudflare_go",
        importpath = "github.com/cloudflare/cloudflare-go",
        sum = "h1:gFqGlGl/5f9UGXAaKapCGUfaTCgRKKnzu2VvzMZlOFA=",
        version = "v0.14.0",
    )

    go_repository(
        name = "com_github_codeclysm_extract_v3",
        importpath = "github.com/codeclysm/extract/v3",
        sum = "h1:sB4LcE3Php7LkhZwN0n2p8GCwZe92PEQutdbGURf5xc=",
        version = "v3.0.2",
    )
    go_repository(
        name = "com_github_consensys_gnark_crypto",
        importpath = "github.com/consensys/gnark-crypto",
        sum = "h1:C43yEtQ6NIf4ftFXD/V55gnGFgPbMQobd//YlnLjUJ8=",
        version = "v0.4.1-0.20210426202927-39ac3d4b3f1f",
    )

    go_repository(
        name = "com_github_containerd_cgroups",
        importpath = "github.com/containerd/cgroups",
        sum = "h1:jN/mbWBEaz+T1pi5OFtnkQ+8qnmEbAr1Oo1FRm5B0dA=",
        version = "v1.0.4",
    )

    go_repository(
        name = "com_github_coreos_go_systemd_v22",
        importpath = "github.com/coreos/go-systemd/v22",
        sum = "h1:y9YHcjnjynCd/DVbg5j9L/33jQM3MxJlbj/zWskzfGU=",
        version = "v22.4.0",
    )

    go_repository(
        name = "com_github_cpuguy83_go_md2man_v2",
        importpath = "github.com/cpuguy83/go-md2man/v2",
        sum = "h1:p1EgwI/C7NhT0JmVkwCD2ZBK8j4aeHQX2pMHHBfMQ6w=",
        version = "v2.0.2",
    )
    go_repository(
        name = "com_github_crackcomm_go_gitignore",
        importpath = "github.com/crackcomm/go-gitignore",
        sum = "h1:HVTnpeuvF6Owjd5mniCL8DEXo7uYXdQEmOP4FJbV5tg=",
        version = "v0.0.0-20170627025303-887ab5e44cc3",
    )

    go_repository(
        name = "com_github_creack_pty",
        importpath = "github.com/creack/pty",
        sum = "h1:uDmaGzcdjhF4i/plgjmEsriH11Y0o7RKapEf/LDaM3w=",
        version = "v1.1.9",
    )

    go_repository(
        name = "com_github_cskr_pubsub",
        importpath = "github.com/cskr/pubsub",
        sum = "h1:vlOzMhl6PFn60gRlTQQsIfVwaPB/B/8MziK8FhEPt/0=",
        version = "v1.0.2",
    )

    go_repository(
        name = "com_github_davecgh_go_spew",
        importpath = "github.com/davecgh/go-spew",
        sum = "h1:vj9j/u1bqnvCEfJOwUhtlOARqs3+rkHYY13jYWTU97c=",
        version = "v1.1.1",
    )
    go_repository(
        name = "com_github_davidlazar_go_crypto",
        importpath = "github.com/davidlazar/go-crypto",
        sum = "h1:pFUpOrbxDR6AkioZ1ySsx5yxlDQZ8stG2b88gTPxgJU=",
        version = "v0.0.0-20200604182044-b73af7476f6c",
    )

    go_repository(
        name = "com_github_deckarep_golang_set",
        importpath = "github.com/deckarep/golang-set",
        sum = "h1:sk9/l/KqpunDwP7pSjUg0keiOOLEnOBHzykLrsPppp4=",
        version = "v1.8.0",
    )

    go_repository(
        name = "com_github_decred_dcrd_crypto_blake256",
        importpath = "github.com/decred/dcrd/crypto/blake256",
        sum = "h1:/8DMNYp9SGi5f0w7uCm6d6M4OU2rGFK09Y2A4Xv7EE0=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_decred_dcrd_dcrec_secp256k1_v4",
        importpath = "github.com/decred/dcrd/dcrec/secp256k1/v4",
        sum = "h1:HbphB4TFFXpv7MNrT52FGrrgVXF1owhMVTHFZIlnvd4=",
        version = "v4.1.0",
    )
    go_repository(
        name = "com_github_deepmap_oapi_codegen",
        importpath = "github.com/deepmap/oapi-codegen",
        sum = "h1:SegyeYGcdi0jLLrpbCMoJxnUUn8GBXHsvr4rbzjuhfU=",
        version = "v1.8.2",
    )

    go_repository(
        name = "com_github_dgraph_io_badger",
        importpath = "github.com/dgraph-io/badger",
        sum = "h1:mNw0qs90GVgGGWylh0umH5iag1j6n/PeJtNvL6KY/x8=",
        version = "v1.6.2",
    )
    go_repository(
        name = "com_github_dgraph_io_badger_v3",
        importpath = "github.com/dgraph-io/badger/v3",
        sum = "h1:dpyM5eCJAtQCBcMCZcT4UBZchuTJgCywerHHgmxfxM8=",
        version = "v3.2103.2",
    )
    go_repository(
        name = "com_github_dgraph_io_ristretto",
        importpath = "github.com/dgraph-io/ristretto",
        sum = "h1:Jv3CGQHp9OjuMBSne1485aDpUkTKEcUqF+jm/LuerPI=",
        version = "v0.1.0",
    )

    go_repository(
        name = "com_github_dgryski_go_rendezvous",
        importpath = "github.com/dgryski/go-rendezvous",
        sum = "h1:lO4WD4F/rVNCu3HqELle0jiPLLBs70cWOduZpkS1E78=",
        version = "v0.0.0-20200823014737-9f7001d12a5f",
    )
    go_repository(
        name = "com_github_dgryski_go_sip13",
        importpath = "github.com/dgryski/go-sip13",
        sum = "h1:RMLoZVzv4GliuWafOuPuQDKSm1SJph7uCRnnS61JAn4=",
        version = "v0.0.0-20181026042036-e10d5fee7954",
    )

    go_repository(
        name = "com_github_dlclark_regexp2",
        importpath = "github.com/dlclark/regexp2",
        sum = "h1:Izz0+t1Z5nI16/II7vuEo/nHjodOg0p7+OiDpjX5t1E=",
        version = "v1.4.1-0.20201116162257-a2a8dda75c91",
    )
    go_repository(
        name = "com_github_docker_docker",
        importpath = "github.com/docker/docker",
        sum = "h1:HlFGsy+9/xrgMmhmN+NGhCc5SHGJ7I+kHosRR1xc/aI=",
        version = "v1.6.2",
    )

    go_repository(
        name = "com_github_docker_go_units",
        importpath = "github.com/docker/go-units",
        sum = "h1:69rxXcBk27SvSaaxTtLh/8llcHD8vYHT7WSdRZ/jvr4=",
        version = "v0.5.0",
    )

    go_repository(
        name = "com_github_dop251_goja",
        importpath = "github.com/dop251/goja",
        sum = "h1:Yt+4K30SdjOkRoRRm3vYNQgR+/ZIy0RmeUDZo7Y8zeQ=",
        version = "v0.0.0-20220405120441-9037c2b61cbf",
    )
    go_repository(
        name = "com_github_dustin_go_humanize",
        importpath = "github.com/dustin/go-humanize",
        sum = "h1:VSnTsYCnlFHaM2/igO1h6X3HA71jcobQuxemgkq4zYo=",
        version = "v1.0.0",
    )

    go_repository(
        name = "com_github_edsrzf_mmap_go",
        importpath = "github.com/edsrzf/mmap-go",
        sum = "h1:6EUwBLQ/Mcr1EYLE4Tn1VdW1A4ckqCQWZBw8Hr0kjpQ=",
        version = "v1.1.0",
    )

    go_repository(
        name = "com_github_elastic_gosigar",
        importpath = "github.com/elastic/gosigar",
        sum = "h1:Dg80n8cr90OZ7x+bAax/QjoW/XqTI11RmA79ZwIm9/4=",
        version = "v0.14.2",
    )

    go_repository(
        name = "com_github_ethereum_go_ethereum",
        importpath = "github.com/ethereum/go-ethereum",
        patch_args = ["-p1"],
        patches = [
            "//third_party:com_github_ethereum_go_ethereum_secp256k1.patch",
        ],
        sum = "h1:i/7d9RBBwiXCEuyduBQzJw/mKmnvzsN14jqBmytw72s=",
        version = "v1.10.26",
    )
    go_repository(
        name = "com_github_facebookgo_atomicfile",
        importpath = "github.com/facebookgo/atomicfile",
        sum = "h1:BBso6MBKW8ncyZLv37o+KNyy0HrrHgfnOaGQC2qvN+A=",
        version = "v0.0.0-20151019160806-2de1f203e7d5",
    )
    go_repository(
        name = "com_github_fatih_color",
        importpath = "github.com/fatih/color",
        sum = "h1:DkWD4oS2D8LGGgTQ6IvwJJXSL5Vp2ffcQg58nFV38Ys=",
        version = "v1.7.0",
    )
    go_repository(
        name = "com_github_fjl_gencodec",
        importpath = "github.com/fjl/gencodec",
        sum = "h1:CndMRAH4JIwxbW8KYq6Q+cGWcGHz0FjGR3QqcInWcW0=",
        version = "v0.0.0-20220412091415-8bb9e558978c",
    )

    go_repository(
        name = "com_github_fjl_memsize",
        importpath = "github.com/fjl/memsize",
        sum = "h1:FtmdgXiUlNeRsoNMFlKLDt+S+6hbjVMEW6RGQ7aUf7c=",
        version = "v0.0.0-20190710130421-bcb5799ab5e5",
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
        sum = "h1:n+5WquG0fcWoWp6xPWfHdbskMCQaFnG6PfBrh1Ky4HY=",
        version = "v1.6.0",
    )
    go_repository(
        name = "com_github_gammazero_deque",
        importpath = "github.com/gammazero/deque",
        sum = "h1:qSdsbG6pgp6nL7A0+K/B7s12mcCY/5l5SIUpMOl+dC0=",
        version = "v0.2.1",
    )
    go_repository(
        name = "com_github_garslo_gogen",
        importpath = "github.com/garslo/gogen",
        sum = "h1:IZqZOB2fydHte3kUgxrzK5E1fW7RQGeDwE8F/ZZnUYc=",
        version = "v0.0.0-20170306192744-1d203ffc1f61",
    )

    go_repository(
        name = "com_github_gballet_go_libpcsclite",
        importpath = "github.com/gballet/go-libpcsclite",
        sum = "h1:tY80oXqGNY4FhTFhk+o9oFHGINQ/+vhlm8HFzi6znCI=",
        version = "v0.0.0-20190607065134-2772fd86a8ff",
    )
    go_repository(
        name = "com_github_go_kit_kit",
        importpath = "github.com/go-kit/kit",
        sum = "h1:Wz+5lgoB0kkuqLEc6NVmwRknTKP6dTGbSqvhZtBI/j0=",
        version = "v0.8.0",
    )

    go_repository(
        name = "com_github_go_logfmt_logfmt",
        importpath = "github.com/go-logfmt/logfmt",
        sum = "h1:otpy5pqBCBZ1ng9RQ0dPu4PN7ba75Y/aA+UpowDyNVA=",
        version = "v0.5.1",
    )
    go_repository(
        name = "com_github_go_logr_logr",
        importpath = "github.com/go-logr/logr",
        sum = "h1:2DntVwHkVopvECVRSlL5PSo9eG+cAkDCuckLubN+rq0=",
        version = "v1.2.3",
    )
    go_repository(
        name = "com_github_go_logr_stdr",
        importpath = "github.com/go-logr/stdr",
        sum = "h1:hSWxHoqTgW2S2qGc0LTAI563KZ5YKYRhT3MFKZMbjag=",
        version = "v1.2.2",
    )

    go_repository(
        name = "com_github_go_ole_go_ole",
        importpath = "github.com/go-ole/go-ole",
        sum = "h1:/Fpf6oFPoeFik9ty7siob0G6Ke8QvQEuVcuChpwXzpY=",
        version = "v1.2.6",
    )

    go_repository(
        name = "com_github_go_redis_redis_v8",
        importpath = "github.com/go-redis/redis/v8",
        sum = "h1:kHoYkfZP6+pe04aFTnhDH6GDROa5yJdHJVNxV3F46Tg=",
        version = "v8.11.4",
    )

    go_repository(
        name = "com_github_go_sourcemap_sourcemap",
        importpath = "github.com/go-sourcemap/sourcemap",
        sum = "h1:W1iEw64niKVGogNgBN3ePyLFfuisuzeidWPMPWmECqU=",
        version = "v2.1.3+incompatible",
    )
    go_repository(
        name = "com_github_go_stack_stack",
        importpath = "github.com/go-stack/stack",
        sum = "h1:ntEHSVwIt7PNXNpgPmVfMrNhLtgjlmnZha2kOpuRiDw=",
        version = "v1.8.1",
    )
    go_repository(
        name = "com_github_go_task_slim_sprig",
        importpath = "github.com/go-task/slim-sprig",
        sum = "h1:p104kn46Q8WdvHunIJ9dAyjPVtrBPhSr3KT2yUst43I=",
        version = "v0.0.0-20210107165309-348f09dbbbc0",
    )
    go_repository(
        name = "com_github_gobwas_httphead",
        importpath = "github.com/gobwas/httphead",
        sum = "h1:exrUm0f4YX0L7EBwZHuCF4GDp8aJfVeBrlLQrs6NqWU=",
        version = "v0.1.0",
    )
    go_repository(
        name = "com_github_gobwas_pool",
        importpath = "github.com/gobwas/pool",
        sum = "h1:xfeeEhW7pwmX8nuLVlqbzVc7udMDrwetjEv+TZIz1og=",
        version = "v0.2.1",
    )
    go_repository(
        name = "com_github_gobwas_ws",
        importpath = "github.com/gobwas/ws",
        sum = "h1:7RFti/xnNkMJnrK7D1yQ/iCIB5OrrY/54/H930kIbHA=",
        version = "v1.1.0",
    )
    go_repository(
        name = "com_github_gobwas_ws_examples",
        importpath = "github.com/gobwas/ws-examples",
        sum = "h1:XC9N1eiAyO1zg62dpOU8bex8emB/zluUtKcbLNjJxGI=",
        version = "v0.0.0-20190625122829-a9e8908d9484",
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
        name = "com_github_golang_glog",
        importpath = "github.com/golang/glog",
        sum = "h1:nfP3RFugxnNRyKgeWd4oI1nYvXpxrx8ck8ZrcizshdQ=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_golang_groupcache",
        importpath = "github.com/golang/groupcache",
        sum = "h1:oI5xCqsCo564l8iNU+DwB5epxmsaqB+rhGL0m5jtYqE=",
        version = "v0.0.0-20210331224755-41bb18bfe9da",
    )

    go_repository(
        name = "com_github_golang_jwt_jwt_v4",
        importpath = "github.com/golang-jwt/jwt/v4",
        sum = "h1:kHL1vqdqWNfATmA0FNMdmZNMyZI1U6O31X4rlIPoBog=",
        version = "v4.3.0",
    )
    go_repository(
        name = "com_github_golang_mock",
        importpath = "github.com/golang/mock",
        sum = "h1:ErTB+efbowRARo13NNdxyJji2egdxLGQhRaY+DUumQc=",
        version = "v1.6.0",
    )

    go_repository(
        name = "com_github_golang_protobuf",
        importpath = "github.com/golang/protobuf",
        sum = "h1:ROPKBNFfQgOUMifHyP+KYbvpjbdoFNs+aK7DXlji0Tw=",
        version = "v1.5.2",
    )
    go_repository(
        name = "com_github_golang_snappy",
        importpath = "github.com/golang/snappy",
        sum = "h1:yAGX7huGHXlcLOEtBnF4w7FQwA26wojNCwOYAEhLjQM=",
        version = "v0.0.4",
    )

    go_repository(
        name = "com_github_google_flatbuffers",
        importpath = "github.com/google/flatbuffers",
        sum = "h1:MVlul7pQNoDzWRLTw5imwYsl+usrS1TXG2H4jg6ImGw=",
        version = "v1.12.1",
    )

    go_repository(
        name = "com_github_google_go_cmp",
        importpath = "github.com/google/go-cmp",
        sum = "h1:xsAVV57WRhGj6kEIi8ReJzQlHHqcBYCElAvkovg3B/4=",
        version = "v0.4.0",
    )

    go_repository(
        name = "com_github_google_gofuzz",
        importpath = "github.com/google/gofuzz",
        sum = "h1:Q75Upo5UN4JbPFURXZ8nLKYUvF85dyFRop/vQ0Rv+64=",
        version = "v1.1.1-0.20200604201612-c04b05f3adfa",
    )
    go_repository(
        name = "com_github_google_gopacket",
        importpath = "github.com/google/gopacket",
        sum = "h1:ves8RnFZPGiFnTS0uPQStjwru6uO6h+nlr9j6fL7kF8=",
        version = "v1.1.19",
    )

    go_repository(
        name = "com_github_google_uuid",
        importpath = "github.com/google/uuid",
        sum = "h1:t6JiXgmwXMjEs8VusXIJk2BXHsn+wx8BZdTaoZ5fu7I=",
        version = "v1.3.0",
    )

    go_repository(
        name = "com_github_gorilla_websocket",
        importpath = "github.com/gorilla/websocket",
        sum = "h1:PPwGk2jz7EePpoHN/+ClbZu8SPxiqlu12wZP/3sWmnc=",
        version = "v1.5.0",
    )
    go_repository(
        name = "com_github_graph_gophers_graphql_go",
        importpath = "github.com/graph-gophers/graphql-go",
        sum = "h1:Eb9x/q6MFpCLz7jBCiP/WTxjSDrYLR1QY41SORZyNJ0=",
        version = "v1.3.0",
    )
    go_repository(
        name = "com_github_grpc_ecosystem_grpc_gateway_v2",
        importpath = "github.com/grpc-ecosystem/grpc-gateway/v2",
        sum = "h1:BZHcxBETFHIdVyhyEfOvn/RdU/QGdLI4y34qQGjGWO0=",
        version = "v2.7.0",
    )

    go_repository(
        name = "com_github_h2non_filetype",
        importpath = "github.com/h2non/filetype",
        sum = "h1:g84/+gdkAT1hnYO+tHpCLoikm13Ju55OkN4KCb1uGEQ=",
        version = "v1.0.6",
    )
    go_repository(
        name = "com_github_hannahhoward_go_pubsub",
        importpath = "github.com/hannahhoward/go-pubsub",
        sum = "h1:3YKHER4nmd7b5qy5t0GWDTwSn4OyRgfAXSmo6VnryBY=",
        version = "v0.0.0-20200423002714-8d62886cc36e",
    )
    go_repository(
        name = "com_github_hashicorp_errwrap",
        importpath = "github.com/hashicorp/errwrap",
        sum = "h1:OxrOeh75EUXMY8TBjag2fzXGZ40LB6IKw45YeGUDY2I=",
        version = "v1.1.0",
    )

    go_repository(
        name = "com_github_hashicorp_go_bexpr",
        importpath = "github.com/hashicorp/go-bexpr",
        sum = "h1:9kuI5PFotCboP3dkDYFr/wi0gg0QVbSNz5oFRpxn4uE=",
        version = "v0.1.10",
    )
    go_repository(
        name = "com_github_hashicorp_go_multierror",
        importpath = "github.com/hashicorp/go-multierror",
        sum = "h1:H5DkEtf6CXdFp0N0Em5UCwQpXMWke8IA0+lD48awMYo=",
        version = "v1.1.1",
    )

    go_repository(
        name = "com_github_hashicorp_golang_lru",
        importpath = "github.com/hashicorp/golang-lru",
        sum = "h1:dg1dEPuWpEqDnvIw251EVy4zlP8gWbsGj4BsUKCRpYs=",
        version = "v0.5.5-0.20210104140557-80c98217689d",
    )
    go_repository(
        name = "com_github_hashicorp_golang_lru_v2",
        importpath = "github.com/hashicorp/golang-lru/v2",
        sum = "h1:5pv5N1lT1fjLg2VQ5KWc7kmucp2x/kvFOnxuVTqZ6x4=",
        version = "v2.0.1",
    )

    go_repository(
        name = "com_github_holiman_bloomfilter_v2",
        importpath = "github.com/holiman/bloomfilter/v2",
        sum = "h1:73e0e/V0tCydx14a0SCYS/EWCxgwLZ18CZcZKVu0fao=",
        version = "v2.0.3",
    )
    go_repository(
        name = "com_github_holiman_uint256",
        importpath = "github.com/holiman/uint256",
        sum = "h1:TXKcSGc2WaxPD2+bmzAsVthL4+pEN0YwXcL5qED83vk=",
        version = "v1.2.2",
    )
    go_repository(
        name = "com_github_hpcloud_tail",
        importpath = "github.com/hpcloud/tail",
        sum = "h1:nfCOvKYfkgYP8hkirhJocXT2+zOD8yUNjXaWfTlyFKI=",
        version = "v1.0.0",
    )

    go_repository(
        name = "com_github_huin_goupnp",
        importpath = "github.com/huin/goupnp",
        sum = "h1:gEe0Dp/lZmPZiDFzJJaOfUpOvv2MKUkoBX8lDrn9vKU=",
        version = "v1.1.0",
    )
    go_repository(
        name = "com_github_influxdata_influxdb",
        importpath = "github.com/influxdata/influxdb",
        sum = "h1:WEypI1BQFTT4teLM+1qkEcvUi0dAvopAI/ir0vAiBg8=",
        version = "v1.8.3",
    )
    go_repository(
        name = "com_github_influxdata_influxdb_client_go_v2",
        importpath = "github.com/influxdata/influxdb-client-go/v2",
        sum = "h1:HGBfZYStlx3Kqvsv1h2pJixbCl/jhnFtxpKFAv9Tu5k=",
        version = "v2.4.0",
    )
    go_repository(
        name = "com_github_influxdata_line_protocol",
        importpath = "github.com/influxdata/line-protocol",
        sum = "h1:vilfsDSy7TDxedi9gyBkMvAirat/oRcL0lFdJBf6tdM=",
        version = "v0.0.0-20210311194329-9aa0e372d097",
    )

    go_repository(
        name = "com_github_ipfs_bbloom",
        importpath = "github.com/ipfs/bbloom",
        sum = "h1:Gi+8EGJ2y5qiD5FbsbpX/TMNcJw8gSqr7eyjHa4Fhvs=",
        version = "v0.0.4",
    )
    go_repository(
        name = "com_github_ipfs_go_bitfield",
        importpath = "github.com/ipfs/go-bitfield",
        sum = "h1:y/XHm2GEmD9wKngheWNNCNL0pzrWXZwCdQGv1ikXknQ=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_ipfs_go_bitswap",
        importpath = "github.com/ipfs/go-bitswap",
        sum = "h1:B81RIwkTnIvSYT1ZCzxjYTeF0Ek88xa9r1AMpTfk+9Q=",
        version = "v0.10.2",
    )
    go_repository(
        name = "com_github_ipfs_go_block_format",
        importpath = "github.com/ipfs/go-block-format",
        sum = "h1:r8t66QstRp/pd/or4dpnbVfXT5Gt7lOqRvC+/dDTpMc=",
        version = "v0.0.3",
    )
    go_repository(
        name = "com_github_ipfs_go_blockservice",
        importpath = "github.com/ipfs/go-blockservice",
        sum = "h1:7MUijAW5SqdsqEW/EhnNFRJXVF8mGU5aGhZ3CQaCWbY=",
        version = "v0.4.0",
    )
    go_repository(
        name = "com_github_ipfs_go_cid",
        importpath = "github.com/ipfs/go-cid",
        sum = "h1:OGgOd+JCFM+y1DjWPmVH+2/4POtpDzwcr7VgnB7mZXc=",
        version = "v0.3.2",
    )
    go_repository(
        name = "com_github_ipfs_go_cidutil",
        importpath = "github.com/ipfs/go-cidutil",
        sum = "h1:RW5hO7Vcf16dplUU60Hs0AKDkQAVPVplr7lk97CFL+Q=",
        version = "v0.1.0",
    )
    go_repository(
        name = "com_github_ipfs_go_datastore",
        importpath = "github.com/ipfs/go-datastore",
        sum = "h1:JKyz+Gvz1QEZw0LsX1IBn+JFCJQH4SJVFtM4uWU0Myk=",
        version = "v0.6.0",
    )
    go_repository(
        name = "com_github_ipfs_go_delegated_routing",
        importpath = "github.com/ipfs/go-delegated-routing",
        sum = "h1:+M1siyTB2H4mHzEnbWjepQxlmKbapVWdbYLexSDODpg=",
        version = "v0.6.0",
    )
    go_repository(
        name = "com_github_ipfs_go_ds_badger",
        importpath = "github.com/ipfs/go-ds-badger",
        sum = "h1:xREL3V0EH9S219kFFueOYJJTcjgNSZ2HY1iSvN7U1Ro=",
        version = "v0.3.0",
    )
    go_repository(
        name = "com_github_ipfs_go_ds_flatfs",
        importpath = "github.com/ipfs/go-ds-flatfs",
        sum = "h1:ZCIO/kQOS/PSh3vcF1H6a8fkRGS7pOfwfPdx4n/KJH4=",
        version = "v0.5.1",
    )
    go_repository(
        name = "com_github_ipfs_go_ds_leveldb",
        importpath = "github.com/ipfs/go-ds-leveldb",
        sum = "h1:s++MEBbD3ZKc9/8/njrn4flZLnCuY9I79v94gBUNumo=",
        version = "v0.5.0",
    )
    go_repository(
        name = "com_github_ipfs_go_ds_measure",
        importpath = "github.com/ipfs/go-ds-measure",
        sum = "h1:sG4goQe0KDTccHMyT45CY1XyUbxe5VwTKpg2LjApYyQ=",
        version = "v0.2.0",
    )
    go_repository(
        name = "com_github_ipfs_go_fetcher",
        importpath = "github.com/ipfs/go-fetcher",
        sum = "h1:UFuRVYX5AIllTiRhi5uK/iZkfhSpBCGX7L70nSZEmK8=",
        version = "v1.6.1",
    )
    go_repository(
        name = "com_github_ipfs_go_filestore",
        importpath = "github.com/ipfs/go-filestore",
        sum = "h1:O2wg7wdibwxkEDcl7xkuQsPvJFRBVgVSsOJ/GP6z3yU=",
        version = "v1.2.0",
    )
    go_repository(
        name = "com_github_ipfs_go_fs_lock",
        importpath = "github.com/ipfs/go-fs-lock",
        sum = "h1:6BR3dajORFrFTkb5EpCUFIAypsoxpGpDSVUdFwzgL9U=",
        version = "v0.0.7",
    )
    go_repository(
        name = "com_github_ipfs_go_graphsync",
        importpath = "github.com/ipfs/go-graphsync",
        sum = "h1:lWiP/WLycoPUYyj3IDEi1GJNP30kFuYOvimcfeuZyQs=",
        version = "v0.13.1",
    )
    go_repository(
        name = "com_github_ipfs_go_ipfs_blockstore",
        importpath = "github.com/ipfs/go-ipfs-blockstore",
        sum = "h1:n3WTeJ4LdICWs/0VSfjHrlqpPpl6MZ+ySd3j8qz0ykw=",
        version = "v1.2.0",
    )
    go_repository(
        name = "com_github_ipfs_go_ipfs_chunker",
        importpath = "github.com/ipfs/go-ipfs-chunker",
        sum = "h1:ojCf7HV/m+uS2vhUGWcogIIxiO5ubl5O57Q7NapWLY8=",
        version = "v0.0.5",
    )
    go_repository(
        name = "com_github_ipfs_go_ipfs_delay",
        importpath = "github.com/ipfs/go-ipfs-delay",
        sum = "h1:r/UXYyRcddO6thwOnhiznIAiSvxMECGgtv35Xs1IeRQ=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_ipfs_go_ipfs_ds_help",
        importpath = "github.com/ipfs/go-ipfs-ds-help",
        sum = "h1:yLE2w9RAsl31LtfMt91tRZcrx+e61O5mDxFRR994w4Q=",
        version = "v1.1.0",
    )
    go_repository(
        name = "com_github_ipfs_go_ipfs_exchange_interface",
        importpath = "github.com/ipfs/go-ipfs-exchange-interface",
        sum = "h1:8lMSJmKogZYNo2jjhUs0izT+dck05pqUw4mWNW9Pw6Y=",
        version = "v0.2.0",
    )
    go_repository(
        name = "com_github_ipfs_go_ipfs_exchange_offline",
        importpath = "github.com/ipfs/go-ipfs-exchange-offline",
        sum = "h1:c/Dg8GDPzixGd0MC8Jh6mjOwU57uYokgWRFidfvEkuA=",
        version = "v0.3.0",
    )
    go_repository(
        name = "com_github_ipfs_go_ipfs_files",
        importpath = "github.com/ipfs/go-ipfs-files",
        sum = "h1:/MbEowmpLo9PJTEQk16m9rKzUHjeP4KRU9nWJyJO324=",
        version = "v0.1.1",
    )
    go_repository(
        name = "com_github_ipfs_go_ipfs_keystore",
        importpath = "github.com/ipfs/go-ipfs-keystore",
        sum = "h1:Fa9xg9IFD1VbiZtrNLzsD0GuELVHUFXCWF64kCPfEXU=",
        version = "v0.0.2",
    )
    go_repository(
        name = "com_github_ipfs_go_ipfs_pinner",
        importpath = "github.com/ipfs/go-ipfs-pinner",
        sum = "h1:kw9hiqh2p8TatILYZ3WAfQQABby7SQARdrdA+5Z5QfY=",
        version = "v0.2.1",
    )
    go_repository(
        name = "com_github_ipfs_go_ipfs_posinfo",
        importpath = "github.com/ipfs/go-ipfs-posinfo",
        sum = "h1:Esoxj+1JgSjX0+ylc0hUmJCOv6V2vFoZiETLR6OtpRs=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_ipfs_go_ipfs_pq",
        importpath = "github.com/ipfs/go-ipfs-pq",
        sum = "h1:e1vOOW6MuOwG2lqxcLA+wEn93i/9laCY8sXAw76jFOY=",
        version = "v0.0.2",
    )
    go_repository(
        name = "com_github_ipfs_go_ipfs_provider",
        importpath = "github.com/ipfs/go-ipfs-provider",
        sum = "h1:eKToBUAb6ZY8iiA6AYVxzW4G1ep67XUaaEBUIYpxhfw=",
        version = "v0.7.1",
    )
    go_repository(
        name = "com_github_ipfs_go_ipfs_routing",
        importpath = "github.com/ipfs/go-ipfs-routing",
        sum = "h1:E+whHWhJkdN9YeoHZNj5itzc+OR292AJ2uE9FFiW0BY=",
        version = "v0.2.1",
    )
    go_repository(
        name = "com_github_ipfs_go_ipfs_util",
        importpath = "github.com/ipfs/go-ipfs-util",
        sum = "h1:59Sswnk1MFaiq+VcaknX7aYEyGyGDAA73ilhEK2POp8=",
        version = "v0.0.2",
    )
    go_repository(
        name = "com_github_ipfs_go_ipld_cbor",
        importpath = "github.com/ipfs/go-ipld-cbor",
        sum = "h1:ovz4CHKogtG2KB/h1zUp5U0c/IzZrL435rCh5+K/5G8=",
        version = "v0.0.5",
    )
    go_repository(
        name = "com_github_ipfs_go_ipld_format",
        importpath = "github.com/ipfs/go-ipld-format",
        sum = "h1:yqJSaJftjmjc9jEOFYlpkwOLVKv68OD27jFLlSghBlQ=",
        version = "v0.4.0",
    )
    go_repository(
        name = "com_github_ipfs_go_ipld_git",
        importpath = "github.com/ipfs/go-ipld-git",
        sum = "h1:TWGnZjS0htmEmlMFEkA3ogrNCqWjIxwr16x1OsdhG+Y=",
        version = "v0.1.1",
    )
    go_repository(
        name = "com_github_ipfs_go_ipld_legacy",
        importpath = "github.com/ipfs/go-ipld-legacy",
        sum = "h1:BvD8PEuqwBHLTKqlGFTHSwrwFOMkVESEvwIYwR2cdcc=",
        version = "v0.1.1",
    )
    go_repository(
        name = "com_github_ipfs_go_ipns",
        importpath = "github.com/ipfs/go-ipns",
        sum = "h1:ai791nTgVo+zTuq2bLvEGmWP1M0A6kGTXUsgv/Yq67A=",
        version = "v0.3.0",
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
        name = "com_github_ipfs_go_merkledag",
        importpath = "github.com/ipfs/go-merkledag",
        sum = "h1:N3yrqSre/ffvdwtHL4MXy0n7XH+VzN8DlzDrJySPa94=",
        version = "v0.8.1",
    )
    go_repository(
        name = "com_github_ipfs_go_metrics_interface",
        importpath = "github.com/ipfs/go-metrics-interface",
        sum = "h1:j+cpbjYvu4R8zbleSs36gvB7jR+wsL2fGD6n0jO4kdg=",
        version = "v0.0.1",
    )
    go_repository(
        name = "com_github_ipfs_go_mfs",
        importpath = "github.com/ipfs/go-mfs",
        sum = "h1:5jz8+ukAg/z6jTkollzxGzhkl3yxm022Za9f2nL5ab8=",
        version = "v0.2.1",
    )
    go_repository(
        name = "com_github_ipfs_go_namesys",
        importpath = "github.com/ipfs/go-namesys",
        sum = "h1:vZEkdqxRiSnxBBJrvYTkwHYBFgibGUSpNtg9BHRyN+o=",
        version = "v0.5.0",
    )
    go_repository(
        name = "com_github_ipfs_go_path",
        importpath = "github.com/ipfs/go-path",
        sum = "h1:tkjga3MtpXyM5v+3EbRvOHEoo+frwi4oumw5K+KYWyA=",
        version = "v0.3.0",
    )
    go_repository(
        name = "com_github_ipfs_go_peertaskqueue",
        importpath = "github.com/ipfs/go-peertaskqueue",
        sum = "h1:7PLjon3RZwRQMgOTvYccZ+mjzkmds/7YzSWKFlBAypE=",
        version = "v0.7.1",
    )
    go_repository(
        name = "com_github_ipfs_go_unixfs",
        importpath = "github.com/ipfs/go-unixfs",
        sum = "h1:qSyyxfB/OiDdWHYiSbyaqKC7zfSE/TFL0QdwkRjBm20=",
        version = "v0.4.0",
    )
    go_repository(
        name = "com_github_ipfs_go_unixfsnode",
        importpath = "github.com/ipfs/go-unixfsnode",
        sum = "h1:9BUxHBXrbNi8mWHc6j+5C580WJqtVw9uoeEKn4tMhwA=",
        version = "v1.4.0",
    )
    go_repository(
        name = "com_github_ipfs_go_verifcid",
        importpath = "github.com/ipfs/go-verifcid",
        sum = "h1:XPnUv0XmdH+ZIhLGKg6U2vaPaRDXb9urMyNVCE7uvTs=",
        version = "v0.0.2",
    )
    go_repository(
        name = "com_github_ipfs_interface_go_ipfs_core",
        importpath = "github.com/ipfs/interface-go-ipfs-core",
        sum = "h1:7tb+2upz8oCcjIyjo1atdMk+P+u7wPmI+GksBlLE8js=",
        version = "v0.7.0",
    )
    go_repository(
        name = "com_github_ipfs_kubo",
        importpath = "github.com/ipfs/kubo",
        sum = "h1:goK1zdbHHTYVczj20BvAoA6PWiYvBdDgk8zDH+bKwUs=",
        version = "v0.16.0",
    )
    go_repository(
        name = "com_github_ipld_edelweiss",
        importpath = "github.com/ipld/edelweiss",
        sum = "h1:KfAZBP8eeJtrLxLhi7r3N0cBCo7JmwSRhOJp3WSpNjk=",
        version = "v0.2.0",
    )
    go_repository(
        name = "com_github_ipld_go_codec_dagpb",
        importpath = "github.com/ipld/go-codec-dagpb",
        sum = "h1:RspDRdsJpLfgCI0ONhTAnbHdySGD4t+LHSPK4X1+R0k=",
        version = "v1.5.0",
    )
    go_repository(
        name = "com_github_ipld_go_ipld_prime",
        importpath = "github.com/ipld/go-ipld-prime",
        sum = "h1:xUk7NUBSWHEXdjiOu2sLXouFJOMs0yoYzeI5RAqhYQo=",
        version = "v0.18.0",
    )

    go_repository(
        name = "com_github_jackpal_go_nat_pmp",
        importpath = "github.com/jackpal/go-nat-pmp",
        sum = "h1:KzKSgb7qkJvOUTqYl9/Hg/me3pWgBmERKrTGD7BdWus=",
        version = "v1.0.2",
    )
    go_repository(
        name = "com_github_jbenet_go_temp_err_catcher",
        importpath = "github.com/jbenet/go-temp-err-catcher",
        sum = "h1:zpb3ZH6wIE8Shj2sKS+khgRvf7T7RABoLk/+KKHggpk=",
        version = "v0.1.0",
    )
    go_repository(
        name = "com_github_jbenet_goprocess",
        importpath = "github.com/jbenet/goprocess",
        sum = "h1:DRGOFReOMqqDNXwW70QkacFW0YN9QnwLV0Vqk+3oU0o=",
        version = "v0.1.4",
    )
    go_repository(
        name = "com_github_jedisct1_go_minisign",
        importpath = "github.com/jedisct1/go-minisign",
        sum = "h1:UvSe12bq+Uj2hWd8aOlwPmoZ+CITRFrdit+sDGfAg8U=",
        version = "v0.0.0-20190909160543-45766022959e",
    )

    go_repository(
        name = "com_github_jmespath_go_jmespath",
        importpath = "github.com/jmespath/go-jmespath",
        sum = "h1:BEgLn5cpjn8UN1mAw4NjwDrS35OdebyEtFe+9YPoQUg=",
        version = "v0.4.0",
    )

    go_repository(
        name = "com_github_juju_errors",
        importpath = "github.com/juju/errors",
        sum = "h1:rhqTjzJlm7EbkELJDKMTU7udov+Se0xZkWmugr6zGok=",
        version = "v0.0.0-20181118221551-089d3ea4e4d5",
    )

    go_repository(
        name = "com_github_julienschmidt_httprouter",
        importpath = "github.com/julienschmidt/httprouter",
        sum = "h1:TDTW5Yz1mjftljbcKqRcrYhd4XeOoI98t+9HbQbYf7g=",
        version = "v1.2.0",
    )
    go_repository(
        name = "com_github_karalabe_usb",
        importpath = "github.com/karalabe/usb",
        sum = "h1:M6QQBNxF+CQ8OFvxrT90BA0qBOXymndZnk5q235mFc4=",
        version = "v0.0.2",
    )

    go_repository(
        name = "com_github_klauspost_compress",
        importpath = "github.com/klauspost/compress",
        sum = "h1:Ai8UzuomSCDw90e1qNMtb15msBXsNpH6gzkkENQNcJo=",
        version = "v1.15.10",
    )

    go_repository(
        name = "com_github_klauspost_cpuid_v2",
        importpath = "github.com/klauspost/cpuid/v2",
        sum = "h1:t0wUqjowdm8ezddV5k0tLWVklVuvLJpoHeb4WBdydm0=",
        version = "v2.1.1",
    )
    go_repository(
        name = "com_github_knadh_koanf",
        importpath = "github.com/knadh/koanf",
        sum = "h1:/k0Bh49SqLyLNfte9r6cvuZWrApOQhglOmhIU3L/zDw=",
        version = "v1.4.0",
    )

    go_repository(
        name = "com_github_koron_go_ssdp",
        importpath = "github.com/koron/go-ssdp",
        sum = "h1:JivLMY45N76b4p/vsWGOKewBQu6uf39y8l+AQ7sDKx8=",
        version = "v0.0.3",
    )
    go_repository(
        name = "com_github_kr_logfmt",
        importpath = "github.com/kr/logfmt",
        sum = "h1:T+h1c/A9Gawja4Y9mFVWj2vyii2bbUNDw3kt9VxK2EY=",
        version = "v0.0.0-20140226030751-b84e30acd515",
    )

    go_repository(
        name = "com_github_kr_pretty",
        importpath = "github.com/kr/pretty",
        sum = "h1:flRD4NNwYAUpkphVc1HcthR4KEIFJ65n8Mw5qdRn3LE=",
        version = "v0.3.1",
    )
    go_repository(
        name = "com_github_kr_pty",
        importpath = "github.com/kr/pty",
        sum = "h1:VkoXIwSboBpnk99O/KFauAEILuNHv5DVFKZMBN/gUgw=",
        version = "v1.1.1",
    )
    go_repository(
        name = "com_github_kr_text",
        importpath = "github.com/kr/text",
        sum = "h1:5Nx0Ya0ZqY2ygV366QzturHI13Jq95ApcVaJBhpS+AY=",
        version = "v0.2.0",
    )
    go_repository(
        name = "com_github_kylelemons_godebug",
        importpath = "github.com/kylelemons/godebug",
        sum = "h1:RPNrshWIDI6G2gRW9EHilWtl7Z6Sb1BR0xunSBf0SNc=",
        version = "v1.1.0",
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
        name = "com_github_libp2p_go_doh_resolver",
        importpath = "github.com/libp2p/go-doh-resolver",
        sum = "h1:gUBa1f1XsPwtpE1du0O+nnZCUqtG7oYi7Bb+0S7FQqw=",
        version = "v0.4.0",
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
        sum = "h1:yqyTeKQJyofWXxEv/eEVUvOrGdt/9x+0PIQ4N1kaxmE=",
        version = "v0.23.2",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_asn_util",
        importpath = "github.com/libp2p/go-libp2p-asn-util",
        sum = "h1:rg3+Os8jbnO5DxkC7K/Utdi+DkY3q/d1/1q+8WeNAsw=",
        version = "v0.2.0",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_core",
        importpath = "github.com/libp2p/go-libp2p-core",
        sum = "h1:fQz4BJyIFmSZAiTbKV8qoYhEH5Dtv/cVhZbG3Ib/+Cw=",
        version = "v0.20.1",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_discovery",
        importpath = "github.com/libp2p/go-libp2p-discovery",
        sum = "h1:6Iu3NyningTb/BmUnEhcTwzwbs4zcywwbfTulM9LHuc=",
        version = "v0.7.0",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_kad_dht",
        importpath = "github.com/libp2p/go-libp2p-kad-dht",
        sum = "h1:akqO3gPMwixR7qFSFq70ezRun97g5hrA/lBW9jrjUYM=",
        version = "v0.18.0",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_kbucket",
        importpath = "github.com/libp2p/go-libp2p-kbucket",
        sum = "h1:spZAcgxifvFZHBD8tErvppbnNiKA5uokDu3CV7axu70=",
        version = "v0.4.7",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_pubsub",
        importpath = "github.com/libp2p/go-libp2p-pubsub",
        sum = "h1:wycbV+f4rreCoVY61Do6g/BUk0RIrbNRcYVbn+QkjGk=",
        version = "v0.6.1",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_pubsub_router",
        importpath = "github.com/libp2p/go-libp2p-pubsub-router",
        sum = "h1:WuYdY42DVIJ+N0qMdq2du/E9poJH+xzsXL7Uptwj9tw=",
        version = "v0.5.0",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_record",
        importpath = "github.com/libp2p/go-libp2p-record",
        sum = "h1:oiNUOCWno2BFuxt3my4i1frNrt7PerzB3queqa1NkQ0=",
        version = "v0.2.0",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_routing_helpers",
        importpath = "github.com/libp2p/go-libp2p-routing-helpers",
        sum = "h1:b7y4aixQ7AwbqYfcOQ6wTw8DQvuRZeTAA0Od3YYN5yc=",
        version = "v0.4.0",
    )
    go_repository(
        name = "com_github_libp2p_go_libp2p_xor",
        importpath = "github.com/libp2p/go-libp2p-xor",
        sum = "h1:hhQwT4uGrBcuAkUGXADuPltalOdpf9aag9kaYNT2tLA=",
        version = "v0.1.0",
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
        sum = "h1:0FpsbsvuSnAhXFnCY0VLFbJOzaK0VnP0r1QT/o4nWRE=",
        version = "v0.2.0",
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
        name = "com_github_libp2p_zeroconf_v2",
        importpath = "github.com/libp2p/zeroconf/v2",
        sum = "h1:Cup06Jv6u81HLhIj1KasuNM/RHHrJ8T7wOTS4+Tv53Q=",
        version = "v2.2.0",
    )
    go_repository(
        name = "com_github_lucas_clemente_quic_go",
        importpath = "github.com/lucas-clemente/quic-go",
        sum = "h1:Z+WMJ++qMLhvpFkRZA+jl3BTxUjm415YBmWanXB8zP0=",
        version = "v0.29.1",
    )

    go_repository(
        name = "com_github_mailru_easygo",
        importpath = "github.com/mailru/easygo",
        sum = "h1:4+gHs0jJFJ06bfN8PshnM6cHcxGjRUVRLo5jndDiKRQ=",
        version = "v0.0.0-20190618140210-3c14a0dc985f",
    )

    go_repository(
        name = "com_github_marten_seemann_qpack",
        importpath = "github.com/marten-seemann/qpack",
        sum = "h1:jvTsT/HpCn2UZJdP+UUB53FfUUgeOyG5K1ns0OJOGVs=",
        version = "v0.2.1",
    )
    go_repository(
        name = "com_github_marten_seemann_qtls_go1_18",
        importpath = "github.com/marten-seemann/qtls-go1-18",
        sum = "h1:JH6jmzbduz0ITVQ7ShevK10Av5+jBEKAHMntXmIV7kM=",
        version = "v0.1.2",
    )
    go_repository(
        name = "com_github_marten_seemann_qtls_go1_19",
        importpath = "github.com/marten-seemann/qtls-go1-19",
        sum = "h1:rLFKD/9mp/uq1SYGYuVZhm83wkmU95pK5df3GufyYYU=",
        version = "v0.1.0",
    )
    go_repository(
        name = "com_github_marten_seemann_tcp",
        importpath = "github.com/marten-seemann/tcp",
        sum = "h1:br0buuQ854V8u83wA0rVZ8ttrq5CpaPZdvrK0LP2lOk=",
        version = "v0.0.0-20210406111302-dfbc87cc63fd",
    )
    go_repository(
        name = "com_github_marten_seemann_webtransport_go",
        importpath = "github.com/marten-seemann/webtransport-go",
        sum = "h1:TnyKp3pEXcDooTaNn4s9dYpMJ7kMnTp7k5h+SgYP/mc=",
        version = "v0.1.1",
    )

    go_repository(
        name = "com_github_mattn_go_colorable",
        importpath = "github.com/mattn/go-colorable",
        sum = "h1:fFA4WZxdEF4tXPZVKMLwD8oUnCTTo08duU7wxecdEvA=",
        version = "v0.1.13",
    )
    go_repository(
        name = "com_github_mattn_go_isatty",
        importpath = "github.com/mattn/go-isatty",
        sum = "h1:bq3VjFmv/sOjHtdEhmkEV4x1AJtvUvOJ2PFAZ5+peKQ=",
        version = "v0.0.16",
    )
    go_repository(
        name = "com_github_mattn_go_pointer",
        importpath = "github.com/mattn/go-pointer",
        sum = "h1:n+XhsuGeVO6MEAp7xyEukFINEa+Quek5psIR/ylA6o0=",
        version = "v0.0.1",
    )

    go_repository(
        name = "com_github_mattn_go_runewidth",
        importpath = "github.com/mattn/go-runewidth",
        sum = "h1:+xnbZSEeDbOIg5/mE6JF0w6n9duR1l3/WmbinWVwUuU=",
        version = "v0.0.14",
    )

    go_repository(
        name = "com_github_matttproud_golang_protobuf_extensions",
        importpath = "github.com/matttproud/golang_protobuf_extensions",
        sum = "h1:4hp9jkHxhMHkqkrB3Ix0jegS5sx/RkqARlsWZ6pIwiU=",
        version = "v1.0.1",
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
        name = "com_github_minio_sha256_simd",
        importpath = "github.com/minio/sha256-simd",
        sum = "h1:v1ta+49hkWZyvaKwrQB8elexRqm6Y0aMLjCNsrYxo6g=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_mitchellh_copystructure",
        importpath = "github.com/mitchellh/copystructure",
        sum = "h1:vpKXTN4ewci03Vljg/q9QvCGUDttBOGBIa15WveJJGw=",
        version = "v1.2.0",
    )
    go_repository(
        name = "com_github_mitchellh_go_homedir",
        importpath = "github.com/mitchellh/go-homedir",
        sum = "h1:lukF9ziXFxDFPkA1vsr5zpc1XuPDn/wFntq5mG+4E0Y=",
        version = "v1.1.0",
    )

    go_repository(
        name = "com_github_mitchellh_mapstructure",
        importpath = "github.com/mitchellh/mapstructure",
        sum = "h1:6h7AQ0yhTcIsmFmnAwQls75jp2Gzs4iB8W7pjMO+rqo=",
        version = "v1.4.2",
    )
    go_repository(
        name = "com_github_mitchellh_pointerstructure",
        importpath = "github.com/mitchellh/pointerstructure",
        sum = "h1:O+i9nHnXS3l/9Wu7r4NrEdwA2VFTicjUEN1uBnDo34A=",
        version = "v1.2.0",
    )
    go_repository(
        name = "com_github_mitchellh_reflectwalk",
        importpath = "github.com/mitchellh/reflectwalk",
        sum = "h1:G2LzWKi524PWgd3mLHV8Y5k7s6XUvT0Gef6zxSIeXaQ=",
        version = "v1.0.2",
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
        sum = "h1:JR6TyF7JjGd3m6FbLU2cOxhC0Li8z8dLNGQ89tUg4F4=",
        version = "v0.1.0",
    )
    go_repository(
        name = "com_github_multiformats_go_multiaddr",
        importpath = "github.com/multiformats/go-multiaddr",
        sum = "h1:gskHcdaCyPtp9XskVwtvEeQOG465sCohbQIirSyqxrc=",
        version = "v0.7.0",
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
        sum = "h1:KhH2kSuCARyuJraYMFxrNO3DqIaYhOdS039kbhgVwpE=",
        version = "v0.6.0",
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
        sum = "h1:gk85QWKxh3TazbLxED/NlDVv8+q+ReFJk7Y2W/KhfNY=",
        version = "v0.0.6",
    )
    go_repository(
        name = "com_github_naoina_go_stringutil",
        importpath = "github.com/naoina/go-stringutil",
        sum = "h1:rCUeRUHjBjGTSHl0VC00jUPLz8/F9dDzYI70Hzifhks=",
        version = "v0.1.0",
    )
    go_repository(
        name = "com_github_naoina_toml",
        importpath = "github.com/naoina/toml",
        sum = "h1:shk/vn9oCoOTmwcouEdwIeOtOGA/ELRUw/GwvxwfT+0=",
        version = "v0.1.2-0.20170918210437-9fafd6967416",
    )

    go_repository(
        name = "com_github_nxadm_tail",
        importpath = "github.com/nxadm/tail",
        sum = "h1:nPr65rt6Y5JFSKQO7qToXr7pePgD6Gwiw05lkbyAQTE=",
        version = "v1.4.8",
    )
    go_repository(
        name = "com_github_oklog_ulid",
        importpath = "github.com/oklog/ulid",
        sum = "h1:EGfNDEx6MqHz8B3uNV6QAib1UR2Lm97sHi3ocA6ESJ4=",
        version = "v1.3.1",
    )

    go_repository(
        name = "com_github_olekukonko_tablewriter",
        importpath = "github.com/olekukonko/tablewriter",
        sum = "h1:P2Ga83D34wi1o9J6Wh1mRuqd4mF/x/lgBS7N7AbDhec=",
        version = "v0.0.5",
    )
    go_repository(
        name = "com_github_oneofone_xxhash",
        importpath = "github.com/OneOfOne/xxhash",
        sum = "h1:KMrpdQIwFcEqXDklaen+P1axHaj9BSKzvpUUfnHldSE=",
        version = "v1.2.2",
    )

    go_repository(
        name = "com_github_onsi_ginkgo",
        importpath = "github.com/onsi/ginkgo",
        sum = "h1:8xi0RTUf59SOSfEtZMvwTvXYMzG4gV23XVHOZiXNtnE=",
        version = "v1.16.5",
    )
    go_repository(
        name = "com_github_onsi_gomega",
        importpath = "github.com/onsi/gomega",
        sum = "h1:o0+MgICZLuZ7xjH7Vx6zS/zcu93/BEp1VwkIW1mEXCE=",
        version = "v1.10.1",
    )
    go_repository(
        name = "com_github_opencontainers_runtime_spec",
        importpath = "github.com/opencontainers/runtime-spec",
        sum = "h1:UfAcuLBJB9Coz72x1hgl8O5RVzTdNiaglX6v2DM6FI0=",
        version = "v1.0.2",
    )

    go_repository(
        name = "com_github_opentracing_opentracing_go",
        importpath = "github.com/opentracing/opentracing-go",
        sum = "h1:uEJPy/1a5RIPAJ0Ov+OIO8OxWu77jEv+1B0VhjKrZUs=",
        version = "v1.2.0",
    )
    go_repository(
        name = "com_github_openzipkin_zipkin_go",
        importpath = "github.com/openzipkin/zipkin-go",
        sum = "h1:CtfRrOVZtbDj8rt1WXjklw0kqqJQwICrCKmlfUuBUUw=",
        version = "v0.4.0",
    )
    go_repository(
        name = "com_github_pbnjay_memory",
        importpath = "github.com/pbnjay/memory",
        sum = "h1:onHthvaw9LFnH4t2DcNVpwGmV9E1BkGknEliJkfwQj0=",
        version = "v0.0.0-20210728143218-7b4eea64cf58",
    )
    go_repository(
        name = "com_github_peterh_liner",
        importpath = "github.com/peterh/liner",
        sum = "h1:oYW+YCJ1pachXTQmzR3rNLYGGz4g/UgFcjb28p/viDM=",
        version = "v1.1.1-0.20190123174540-a2c9a5303de7",
    )

    go_repository(
        name = "com_github_pkg_diff",
        importpath = "github.com/pkg/diff",
        sum = "h1:aoZm08cpOy4WuID//EZDgcC4zIxODThtZNPirFr42+A=",
        version = "v0.0.0-20210226163009-20ebb0f2a09e",
    )

    go_repository(
        name = "com_github_pkg_errors",
        importpath = "github.com/pkg/errors",
        sum = "h1:FEBLx1zS214owpjy7qsBeixbURkuhQAwrK5UwLGTwt4=",
        version = "v0.9.1",
    )
    go_repository(
        name = "com_github_pmezard_go_difflib",
        importpath = "github.com/pmezard/go-difflib",
        sum = "h1:4DBwDE0NGyQoBHbLQYPwSUPoCMWR5BEzIk/f1lZbAQM=",
        version = "v1.0.0",
    )
    go_repository(
        name = "com_github_polydawn_refmt",
        importpath = "github.com/polydawn/refmt",
        sum = "h1:ZOcivgkkFRnjfoTcGsDq3UQYiBmekwLA+qg0OjyB/ls=",
        version = "v0.0.0-20201211092308-30ac6d18308e",
    )

    go_repository(
        name = "com_github_prometheus_client_golang",
        importpath = "github.com/prometheus/client_golang",
        sum = "h1:b71QUfeo5M8gq2+evJdTPfZhYMAU0uKPkyPJ7TPsloU=",
        version = "v1.13.0",
    )
    go_repository(
        name = "com_github_prometheus_client_model",
        importpath = "github.com/prometheus/client_model",
        sum = "h1:uq5h0d+GuxiXLJLNABMgp2qUWDPiLvgCzz2dUR+/W/M=",
        version = "v0.2.0",
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
        name = "com_github_prometheus_tsdb",
        importpath = "github.com/prometheus/tsdb",
        sum = "h1:YZcsG11NqnK4czYLrWd9mpEuAJIHVQLwdrleYfszMAA=",
        version = "v0.7.1",
    )
    go_repository(
        name = "com_github_raulk_go_watchdog",
        importpath = "github.com/raulk/go-watchdog",
        sum = "h1:oUmdlHxdkXRJlwfG0O9omj8ukerm8MEQavSiDTEtBsk=",
        version = "v1.3.0",
    )
    go_repository(
        name = "com_github_rhnvrm_simples3",
        importpath = "github.com/rhnvrm/simples3",
        sum = "h1:H0DJwybR6ryQE+Odi9eqkHuzjYAeJgtGcGtuBwOhsH8=",
        version = "v0.6.1",
    )
    go_repository(
        name = "com_github_rivo_uniseg",
        importpath = "github.com/rivo/uniseg",
        sum = "h1:8TfxU8dW6PdqD27gjM8MVNuicgxIjxpm4K7x4jp8sis=",
        version = "v0.4.4",
    )

    go_repository(
        name = "com_github_rjeczalik_notify",
        importpath = "github.com/rjeczalik/notify",
        sum = "h1:MiTWrPj55mNDHEiIX5YUSKefw/+lCQVoAFmD6oQm5w8=",
        version = "v0.9.2",
    )
    go_repository(
        name = "com_github_rogpeppe_go_internal",
        importpath = "github.com/rogpeppe/go-internal",
        sum = "h1:73kH8U+JUqXU8lRuOHeVHaa/SZPifC7BkcraZVejAe8=",
        version = "v1.9.0",
    )

    go_repository(
        name = "com_github_rs_cors",
        importpath = "github.com/rs/cors",
        sum = "h1:+88SsELBHx5r+hZ8TCkggzSstaWNbDvThkVK8H6f9ik=",
        version = "v1.7.0",
    )

    go_repository(
        name = "com_github_russross_blackfriday_v2",
        importpath = "github.com/russross/blackfriday/v2",
        sum = "h1:JIOH55/0cWyOuilr9/qlrm0BSXldqnqwMsf35Ld67mk=",
        version = "v2.1.0",
    )

    go_repository(
        name = "com_github_shirou_gopsutil",
        importpath = "github.com/shirou/gopsutil",
        sum = "h1:+1+c1VGhc88SSonWP6foOcLhvnKlUeu/erjjvaPEYiI=",
        version = "v3.21.11+incompatible",
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
        name = "com_github_spf13_pflag",
        importpath = "github.com/spf13/pflag",
        sum = "h1:iy+VFUOCP1a+8yFto/drg2CJ5u0yRoB7fZw3DKv/JXA=",
        version = "v1.0.5",
    )

    go_repository(
        name = "com_github_stackexchange_wmi",
        importpath = "github.com/StackExchange/wmi",
        sum = "h1:fLjPD/aNc3UIOA6tDi6QXUemppXK3P9BI7mr2hd6gx8=",
        version = "v0.0.0-20180116203802-5d049714c4a6",
    )
    go_repository(
        name = "com_github_status_im_keycard_go",
        importpath = "github.com/status-im/keycard-go",
        sum = "h1:Gb2Tyox57NRNuZ2d3rmvB3pcmbu7O1RS3m8WRx7ilrg=",
        version = "v0.0.0-20190316090335-8537d3370df4",
    )
    go_repository(
        name = "com_github_stebalien_go_bitfield",
        importpath = "github.com/Stebalien/go-bitfield",
        sum = "h1:X3kbSSPUaJK60wV2hjOPZwmpljr6VGCqdq4cBLhbQBo=",
        version = "v0.0.1",
    )

    go_repository(
        name = "com_github_stretchr_objx",
        importpath = "github.com/stretchr/objx",
        sum = "h1:1zr/of2m5FGMsad5YfcqgdqdWrIhu+EBEJRhR1U7z/c=",
        version = "v0.5.0",
    )
    go_repository(
        name = "com_github_stretchr_testify",
        importpath = "github.com/stretchr/testify",
        sum = "h1:+h33VjcLVPDHtOdpUCuF+7gSuG3yGIftsP1YvFihtJ8=",
        version = "v1.8.2",
    )
    go_repository(
        name = "com_github_supranational_blst",
        importpath = "github.com/supranational/blst",
        sum = "h1:m+8fKfQwCAy1QjzINvKe/pYtLjo2dl59x2w9YSEJxuY=",
        version = "v0.3.8-0.20220526154634-513d2456b344",
    )

    go_repository(
        name = "com_github_syndtr_goleveldb",
        importpath = "github.com/syndtr/goleveldb",
        sum = "h1:epCh84lMvA70Z7CTTCmYQn2CKbY8j86K7/FAIr141uY=",
        version = "v1.0.1-0.20210819022825-2ae1ddf74ef7",
    )

    go_repository(
        name = "com_github_tidwall_gjson",
        importpath = "github.com/tidwall/gjson",
        sum = "h1:6aeJ0bzojgWLa82gDQHcx3S0Lr/O51I9bJ5nv6JFx5w=",
        version = "v1.14.0",
    )
    go_repository(
        name = "com_github_tidwall_match",
        importpath = "github.com/tidwall/match",
        sum = "h1:+Ho715JplO36QYgwN9PGYNhgZvoUSc9X2c80KVTi+GA=",
        version = "v1.1.1",
    )
    go_repository(
        name = "com_github_tidwall_pretty",
        importpath = "github.com/tidwall/pretty",
        sum = "h1:RWIZEg2iJ8/g6fDDYzMpobmaoGh5OLl4AXtGUGPcqCs=",
        version = "v1.2.0",
    )

    go_repository(
        name = "com_github_tklauser_go_sysconf",
        importpath = "github.com/tklauser/go-sysconf",
        sum = "h1:89WgdJhk5SNwJfu+GKyYveZ4IaJ7xAkecBo+KdJV0CM=",
        version = "v0.3.11",
    )
    go_repository(
        name = "com_github_tklauser_numcpus",
        importpath = "github.com/tklauser/numcpus",
        sum = "h1:kebhY2Qt+3U6RNK7UqpYNA+tJ23IBEGKkB7JQBfDYms=",
        version = "v0.6.0",
    )
    go_repository(
        name = "com_github_tyler_smith_go_bip39",
        importpath = "github.com/tyler-smith/go-bip39",
        sum = "h1:wHSqTBrZW24CsNJDfeh9Ex6Pm0Rcpc7qrgKBiL44vF4=",
        version = "v1.0.1-0.20181017060643-dbb3b84ba2ef",
    )

    go_repository(
        name = "com_github_urfave_cli_v2",
        importpath = "github.com/urfave/cli/v2",
        sum = "h1:x3p8awjp/2arX+Nl/G2040AZpOCHS/eMJJ1/a+mye4Y=",
        version = "v2.10.2",
    )

    go_repository(
        name = "com_github_victoriametrics_fastcache",
        importpath = "github.com/VictoriaMetrics/fastcache",
        sum = "h1:i0mICQuojGDL3KblA7wUNlY5lOK6a4bwt3uRKnkZU40=",
        version = "v1.12.1",
    )

    go_repository(
        name = "com_github_whyrusleeping_base32",
        importpath = "github.com/whyrusleeping/base32",
        sum = "h1:BCPnHtcboadS0DvysUuJXZ4lWVv5Bh5i7+tbIyi+ck4=",
        version = "v0.0.0-20170828182744-c30ac30633cc",
    )
    go_repository(
        name = "com_github_whyrusleeping_cbor_gen",
        importpath = "github.com/whyrusleeping/cbor-gen",
        sum = "h1:bsUlNhdmbtlfdLVXAVfuvKQ01RnWAM09TVrJkI7NZs4=",
        version = "v0.0.0-20210219115102-f37d292932f2",
    )
    go_repository(
        name = "com_github_whyrusleeping_chunker",
        importpath = "github.com/whyrusleeping/chunker",
        sum = "h1:jQa4QT2UP9WYv2nzyawpKMOCl+Z/jW7djv2/J50lj9E=",
        version = "v0.0.0-20181014151217-fe64bd25879f",
    )
    go_repository(
        name = "com_github_whyrusleeping_go_keyspace",
        importpath = "github.com/whyrusleeping/go-keyspace",
        sum = "h1:EKhdznlJHPMoKr0XTrX+IlJs1LH3lyx2nfr1dOlZ79k=",
        version = "v0.0.0-20160322163242-5b898ac5add1",
    )
    go_repository(
        name = "com_github_whyrusleeping_multiaddr_filter",
        importpath = "github.com/whyrusleeping/multiaddr-filter",
        sum = "h1:E9S12nwJwEOXe2d6gT6qxdvqMnNq+VnSsKPgm2ZZNds=",
        version = "v0.0.0-20160516205228-e903e4adabd7",
    )
    go_repository(
        name = "com_github_whyrusleeping_timecache",
        importpath = "github.com/whyrusleeping/timecache",
        sum = "h1:lYbXeSvJi5zk5GLKVuid9TVjS9a0OmLIDKTfoZBL6Ow=",
        version = "v0.0.0-20160911033111-cfcb2f1abfee",
    )
    go_repository(
        name = "com_github_wi2l_jsondiff",
        importpath = "github.com/wI2L/jsondiff",
        sum = "h1:dE00WemBa1uCjrzQUUTE/17I6m5qAaN0EMFOg2Ynr/k=",
        version = "v0.2.0",
    )

    go_repository(
        name = "com_github_xrash_smetrics",
        importpath = "github.com/xrash/smetrics",
        sum = "h1:bAn7/zixMGCfxrRTfdpNzjtPYqr8smhKouy9mxVdGPU=",
        version = "v0.0.0-20201216005158-039620a65673",
    )

    go_repository(
        name = "com_github_yuin_goldmark",
        importpath = "github.com/yuin/goldmark",
        sum = "h1:fVcFKWvrslecOb/tg+Cc05dkeYx540o0FuFt3nUVDoE=",
        version = "v1.4.13",
    )
    go_repository(
        name = "com_github_yuin_gopher_lua",
        importpath = "github.com/yuin/gopher-lua",
        sum = "h1:k/gmLsJDWwWqbLCur2yWnJzwQEKRcAHXo6seXGuSwWw=",
        version = "v0.0.0-20210529063254-f4c35e4016d9",
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
        name = "in_gopkg_alecthomas_kingpin_v2",
        importpath = "gopkg.in/alecthomas/kingpin.v2",
        sum = "h1:jMFz6MfLP0/4fUyZle81rXUoxOBFi19VUFKVDOQfozc=",
        version = "v2.2.6",
    )

    go_repository(
        name = "in_gopkg_check_v1",
        importpath = "gopkg.in/check.v1",
        sum = "h1:Hei/4ADfdWqJk1ZMxUNpqntNwaWcugrBjAiHlqqRiVk=",
        version = "v1.0.0-20201130134442-10cb98267c6c",
    )

    go_repository(
        name = "in_gopkg_fsnotify_v1",
        importpath = "gopkg.in/fsnotify.v1",
        sum = "h1:xOHLXZwVvI9hhs+cLKq5+I5onOuwQLhQwiu63xxlHs4=",
        version = "v1.4.7",
    )

    go_repository(
        name = "in_gopkg_natefinch_lumberjack_v2",
        importpath = "gopkg.in/natefinch/lumberjack.v2",
        sum = "h1:1Lc07Kr7qY4U2YPouBjpCLxpiyxIVoxqXgkXLknAOE8=",
        version = "v2.0.0",
    )

    go_repository(
        name = "in_gopkg_natefinch_npipe_v2",
        importpath = "gopkg.in/natefinch/npipe.v2",
        sum = "h1:+JknDZhAj8YMt7GC73Ei8pv4MzjDUNPHgQWJdtMAaDU=",
        version = "v2.0.0-20160621034901-c1b8fa8bdcce",
    )
    go_repository(
        name = "in_gopkg_square_go_jose_v2",
        importpath = "gopkg.in/square/go-jose.v2",
        sum = "h1:7odma5RETjNHWJnR32wx8t+Io4djHE1PqxCFx3iiZ2w=",
        version = "v2.5.1",
    )

    go_repository(
        name = "in_gopkg_tomb_v1",
        importpath = "gopkg.in/tomb.v1",
        sum = "h1:uRGJdciOHaEIrze2W8Q3AKkepLTh2hOroT7a+7czfdQ=",
        version = "v1.0.0-20141024135613-dd632973f1e7",
    )

    go_repository(
        name = "in_gopkg_yaml_v2",
        importpath = "gopkg.in/yaml.v2",
        sum = "h1:D8xgwECY7CYvx+Y2n4sBz93Jn9JRvxdiyyo8CTfuKaY=",
        version = "v2.4.0",
    )
    go_repository(
        name = "in_gopkg_yaml_v3",
        importpath = "gopkg.in/yaml.v3",
        sum = "h1:fxVm/GzAzEWqLHuvctI91KS9hhNmmWOoWu0XTYJS7CA=",
        version = "v3.0.1",
    )
    go_repository(
        name = "io_opencensus_go",
        importpath = "go.opencensus.io",
        sum = "h1:gqCw0LfLxScz8irSi8exQc7fyQ0fKQU/qnC/X8+V/1M=",
        version = "v0.23.0",
    )
    go_repository(
        name = "io_opentelemetry_go_otel",
        importpath = "go.opentelemetry.io/otel",
        sum = "h1:Z2lA3Tdch0iDcrhJXDIlC94XE+bxok1F9B+4Lz/lGsM=",
        version = "v1.7.0",
    )
    go_repository(
        name = "io_opentelemetry_go_otel_exporters_jaeger",
        importpath = "go.opentelemetry.io/otel/exporters/jaeger",
        sum = "h1:wXgjiRldljksZkZrldGVe6XrG9u3kYDyQmkZwmm5dI0=",
        version = "v1.7.0",
    )
    go_repository(
        name = "io_opentelemetry_go_otel_exporters_otlp_internal_retry",
        importpath = "go.opentelemetry.io/otel/exporters/otlp/internal/retry",
        sum = "h1:7Yxsak1q4XrJ5y7XBnNwqWx9amMZvoidCctv62XOQ6Y=",
        version = "v1.7.0",
    )
    go_repository(
        name = "io_opentelemetry_go_otel_exporters_otlp_otlptrace",
        importpath = "go.opentelemetry.io/otel/exporters/otlp/otlptrace",
        sum = "h1:cMDtmgJ5FpRvqx9x2Aq+Mm0O6K/zcUkH73SFz20TuBw=",
        version = "v1.7.0",
    )
    go_repository(
        name = "io_opentelemetry_go_otel_exporters_otlp_otlptrace_otlptracegrpc",
        importpath = "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc",
        sum = "h1:MFAyzUPrTwLOwCi+cltN0ZVyy4phU41lwH+lyMyQTS4=",
        version = "v1.7.0",
    )
    go_repository(
        name = "io_opentelemetry_go_otel_exporters_otlp_otlptrace_otlptracehttp",
        importpath = "go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp",
        sum = "h1:pLP0MH4MAqeTEV0g/4flxw9O8Is48uAIauAnjznbW50=",
        version = "v1.7.0",
    )
    go_repository(
        name = "io_opentelemetry_go_otel_exporters_stdout_stdouttrace",
        importpath = "go.opentelemetry.io/otel/exporters/stdout/stdouttrace",
        sum = "h1:8hPcgCg0rUJiKE6VWahRvjgLUrNl7rW2hffUEPKXVEM=",
        version = "v1.7.0",
    )
    go_repository(
        name = "io_opentelemetry_go_otel_exporters_zipkin",
        importpath = "go.opentelemetry.io/otel/exporters/zipkin",
        sum = "h1:X0FZj+kaIdLi29UiyrEGDhRTYsEXj9GdEW5Y39UQFEE=",
        version = "v1.7.0",
    )
    go_repository(
        name = "io_opentelemetry_go_otel_sdk",
        importpath = "go.opentelemetry.io/otel/sdk",
        sum = "h1:4OmStpcKVOfvDOgCt7UriAPtKolwIhxpnSNI/yK+1B0=",
        version = "v1.7.0",
    )
    go_repository(
        name = "io_opentelemetry_go_otel_trace",
        importpath = "go.opentelemetry.io/otel/trace",
        sum = "h1:O37Iogk1lEkMRXewVtZ1BBTVn5JEp8GrJvP92bJqC6o=",
        version = "v1.7.0",
    )
    go_repository(
        name = "io_opentelemetry_go_proto_otlp",
        importpath = "go.opentelemetry.io/proto/otlp",
        sum = "h1:WHzDWdXUvbc5bG2ObdrGfaNpQz7ft7QN9HHmJlbiB1E=",
        version = "v0.16.0",
    )

    go_repository(
        name = "org_bazil_fuse",
        importpath = "bazil.org/fuse",
        sum = "h1:utDghgcjE8u+EBjHOgYT+dJPcnDF05KqWMBcjuJy510=",
        version = "v0.0.0-20200117225306-7b5117fecadc",
    )
    go_repository(
        name = "org_go4",
        importpath = "go4.org",
        sum = "h1:BNJlw5kRTzdmyfh5U8F93HA2OwkP7ZGwA51eJ/0wKOU=",
        version = "v0.0.0-20200411211856-f5505b9728dd",
    )

    go_repository(
        name = "org_golang_google_genproto",
        importpath = "google.golang.org/genproto",
        sum = "h1:b9mVrqYfq3P4bCdaLg1qtBnPzUYgglsIdjZkL/fQVOE=",
        version = "v0.0.0-20211118181313-81c1377c94b1",
    )
    go_repository(
        name = "org_golang_google_grpc",
        importpath = "google.golang.org/grpc",
        sum = "h1:9n77onPX5F3qfFCqjy9dhn8PbNQsIKeVU04J9G7umt8=",
        version = "v1.47.0",
    )

    go_repository(
        name = "org_golang_google_protobuf",
        importpath = "google.golang.org/protobuf",
        sum = "h1:d0NfwRgPtno5B1Wa6L2DAG+KivqkdutMf1UhdNx175w=",
        version = "v1.28.1",
    )
    go_repository(
        name = "org_golang_x_crypto",
        importpath = "golang.org/x/crypto",
        sum = "h1:AvwMYaRytfdeVt3u6mLaxYtErKYjxA2OXjJ1HHq6t3A=",
        version = "v0.7.0",
    )
    go_repository(
        name = "org_golang_x_exp",
        importpath = "golang.org/x/exp",
        sum = "h1:SCE/18RnFsLrjydh/R/s5EVvHoZprqEQUuoxK8q2Pc4=",
        version = "v0.0.0-20220916125017-b168a2c6b86b",
    )
    go_repository(
        name = "org_golang_x_mobile",
        importpath = "golang.org/x/mobile",
        sum = "h1:4+4C/Iv2U4fMZBiMCc98MG1In4gJY5YRhtpDNeDeHWs=",
        version = "v0.0.0-20190719004257-d2bd2a29d028",
    )

    go_repository(
        name = "org_golang_x_mod",
        importpath = "golang.org/x/mod",
        sum = "h1:6zppjxzCulZykYSLyVDYbneBfbaBIQPYMevg0bEwv2s=",
        version = "v0.6.0-dev.0.20220419223038-86c51ed26bb4",
    )
    go_repository(
        name = "org_golang_x_net",
        importpath = "golang.org/x/net",
        sum = "h1:Zrh2ngAOFYneWTAIAPethzeaQLuHwhuBkuV6ZiRnUaQ=",
        version = "v0.8.0",
    )

    go_repository(
        name = "org_golang_x_sync",
        importpath = "golang.org/x/sync",
        sum = "h1:wsuoTGHzEhffawBOhz5CYhcrV4IdKZbEyZjBMuTp12o=",
        version = "v0.1.0",
    )
    go_repository(
        name = "org_golang_x_sys",
        importpath = "golang.org/x/sys",
        sum = "h1:MVltZSvRTcU2ljQOhs94SXPftV6DCNnZViHeQps87pQ=",
        version = "v0.6.0",
    )
    go_repository(
        name = "org_golang_x_term",
        importpath = "golang.org/x/term",
        sum = "h1:clScbb1cHjoCkyRbWwBEUZ5H/tIFu5TAXIqaZD0Gcjw=",
        version = "v0.6.0",
    )
    go_repository(
        name = "org_golang_x_text",
        importpath = "golang.org/x/text",
        sum = "h1:57P1ETyNKtuIjB4SRd15iJxuhj8Gc416Y78H3qgMh68=",
        version = "v0.8.0",
    )
    go_repository(
        name = "org_golang_x_time",
        importpath = "golang.org/x/time",
        sum = "h1:O8mE0/t419eoIwhTFpKVkHiTs/Igowgfkj25AcZrtiE=",
        version = "v0.0.0-20210220033141-f8bda1e9f3ba",
    )
    go_repository(
        name = "org_golang_x_tools",
        importpath = "golang.org/x/tools",
        replace = "golang.org/x/tools",
        sum = "h1:VveCTK38A2rkS8ZqFY25HIDFscX5X9OoEhJd3quQmXU=",
        version = "v0.1.12",
    )
    go_repository(
        name = "org_golang_x_xerrors",
        importpath = "golang.org/x/xerrors",
        sum = "h1:uF6paiQQebLeSXkrTqHqz0MXhXXS1KgF41eUdBNvxK0=",
        version = "v0.0.0-20220609144429-65e65417b02f",
    )
    go_repository(
        name = "org_uber_go_atomic",
        importpath = "go.uber.org/atomic",
        sum = "h1:9qC72Qh0+3MqyJbAn8YU5xVq1frD8bn3JtD2oXtafVQ=",
        version = "v1.10.0",
    )
    go_repository(
        name = "org_uber_go_dig",
        importpath = "go.uber.org/dig",
        sum = "h1:fyakRgZDdi2F8FgwJJoRGangMSPTIxPSLGzR3Oh0/54=",
        version = "v1.14.1",
    )
    go_repository(
        name = "org_uber_go_fx",
        importpath = "go.uber.org/fx",
        sum = "h1:S42dZ6Pok8hQ3jxKwo6ZMYcCgHQA/wAS/gnpRa1Pksg=",
        version = "v1.17.1",
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
        sum = "h1:OjGQ5KQDEUawVHxNwQgPpiypGHOxo2mNZsOqTak4fFY=",
        version = "v1.23.0",
    )