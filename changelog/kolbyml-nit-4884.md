### Added

- Store chaintip block recordings in a dedicated append only freezer  instead of the key value database, rolling back the freezer head to handle reorgs. Writes are synced so recordings survive unclean shutdowns.
