### Configuration
- Reduced the batch poster's default blob transaction tip further: `--node.batch-poster.data-poster.min-blob-tx-tip-cap-gwei` lowered from 0.01 to 0.001, and `--node.batch-poster.data-poster.blob-tx-replacement-times` changed to `5m,10m,15m,20m,25m,30m,35m,40m,45m,50m,55m,1h,2h,4h,8h,16h` (one tip bump every 5 minutes for the first hour, reaching ~4 gwei after 1 hour in the worst case).

### Added
- New data poster metrics: `arb/dataposter/tip/bumps` (count of replace-by-fee tip bumps), `arb/dataposter/tip/bumps/deferred` (count of scheduled replacements skipped because the new caps did not meet the minimum replace-by-fee increase), `arb/dataposter/tip/last`, `arb/dataposter/fee/last`, and `arb/dataposter/blobfee/last` (tip, fee, and blob fee caps of the last sent transaction, wei), and `arb/dataposter/tip/suggested` (unclamped suggested tip from the parent chain RPC, wei).
