# Migration rollout

The first rollout from schema v2 to v3 must be performed with the old bot
fully stopped. The v2 binary does not acquire the v3 data-directory lock and
can overwrite files while migration is running.

Required order:

1. Stop every v2 bot process that uses the data directory.
2. Make an external copy or snapshot of the whole data directory.
3. Run a read-only check:

   ```sh
   ./lit-night-bot migrate
   ```

4. Apply the migration only after confirming the old process is stopped:

   ```sh
   ./lit-night-bot migrate --apply --confirm-bot-stopped
   ```

5. Verify that the command reports successful v3 files and generated backups
   under `_migration/backups`.
6. Start the v3 bot. Do not start the old v2 binary against migrated files.

If rollback is required, stop the v3 bot first and restore the corresponding
v2 files from the external snapshot or `_migration/backups` before starting
the v2 binary. Never run migration and a bot writer concurrently.
