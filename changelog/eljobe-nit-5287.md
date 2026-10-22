### Fixed

- Stylus program activations are now written to disk in bounded partial batches, and activation database read/write failures are reported instead of crashing the node (port of the v3.11.x db-writes fix).
