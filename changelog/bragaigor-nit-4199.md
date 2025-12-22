### Changed
- Refactor openInitializeChainDb for Execution/Consensus split
- Introduce downloadDB() to download database
- Introduce openExistingExecutionDB() for opening local execution DB
- Introduce openDownloadedExecutionDB() for opening downloaded execution DB
- Introduce getNewBlockchain() for initializing core.BlockChain
- Introduce getInit() initializes data reader and chainConfig from genesis file 
