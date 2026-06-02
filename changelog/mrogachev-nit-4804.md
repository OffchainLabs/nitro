### Added
- NIT-4804: address-filter `HashStore` now supports `hashing_scheme: "sha256-rawbytesinput"` on the S3 hash list payload, hashing `sha256(salt[:] || address[:])` instead of the string-concat form. Empty or `"sha256-stringinput"` keeps the existing behavior; any other value causes the hash list load to fail.
