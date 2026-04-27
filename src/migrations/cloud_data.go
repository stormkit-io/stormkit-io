package migrations

import (
	"database/sql"

	"github.com/stormkit-io/stormkit-io/src/lib/slog"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

func hashLicenseKey(key string) string {
	return utils.SHA256Hash([]byte(key))
}

// DecryptLicenseKeys migrates old licenses whose license_key was stored as an
// AES-GCM encrypted value to plaintext. It is idempotent: decrypting an already
// plaintext key fails AES-GCM authentication and returns "", so those rows are
// skipped automatically.
func DecryptLicenseKeys(db *sql.DB) {
	rows, err := db.Query(`SELECT license_id, license_key FROM skitapi.licenses WHERE user_id IS NOT NULL`)

	if err != nil {
		slog.Errorf("DecryptLicenseKeys: error querying licenses: %v", err)
		return
	}

	defer rows.Close()

	type entry struct {
		id  int64
		key string
	}

	var toUpdate []entry

	for rows.Next() {
		var id int64
		var encryptedKey string

		if err := rows.Scan(&id, &encryptedKey); err != nil {
			slog.Errorf("DecryptLicenseKeys: error scanning row: %v", err)
			continue
		}

		plaintext := utils.DecryptToString(encryptedKey)

		if plaintext == "" {
			continue
		}

		toUpdate = append(toUpdate, entry{id: id, key: hashLicenseKey(plaintext)})
	}

	if err := rows.Err(); err != nil {
		slog.Errorf("DecryptLicenseKeys: error iterating rows: %v", err)
		return
	}

	for _, e := range toUpdate {
		if _, err := db.Exec(`UPDATE skitapi.licenses SET license_key = $1 WHERE license_id = $2`, e.key, e.id); err != nil {
			slog.Errorf("DecryptLicenseKeys: error updating license %d: %v", e.id, err)
		} else {
			slog.Infof("DecryptLicenseKeys: migrated license %d", e.id)
		}
	}

	if len(toUpdate) == 0 {
		slog.Info("DecryptLicenseKeys: nothing to migrate")
	}
}
