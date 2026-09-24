JetStream `max_file_store` / `max_memory_store` strings no longer parse.

A config of `max_file_store: "1K"` used to mean 1024 bytes. An empty string used to mean 0. Integer values still ought to pass through as-is.

After the last change `"1K"` is treated as 0, so a 1K limit becomes unlimited-or-zero depending on the caller, and operators who set `1K`/`1M`/`1G` see the wrong reservation.
