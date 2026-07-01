package idgen

import (
	"database/sql"
	"strconv"

	"github.com/google/uuid"
)

const (
	UserUIDStart       = int64(10000000)
	RoomRIDStart       = int64(20000000)
	ReservedSuperUID   = "66666666"
	ReservedSuperEmail = "gang-chat@outlook.com"
	ReservedSuperName  = "GANG"
)

func New(prefix string) string {
	return prefix + "_" + uuid.NewString()
}

// NextUserUID atomically allocates the next human-facing user uid.
func NextUserUID(db *sql.DB) string {
	return nextSeq(db, "user_uid", UserUIDStart)
}

// NextRoomRID atomically allocates the next human-facing room rid.
func NextRoomRID(db *sql.DB) string {
	return nextSeq(db, "room_rid", RoomRIDStart)
}

// nextSeq atomically allocates and returns the next value for the named
// sequence. The row is locked with SELECT ... FOR UPDATE, so concurrent callers
// cannot receive the same id.
func nextSeq(db *sql.DB, name string, start int64) string {
	allocated := start
	tx, err := db.Begin()
	if err != nil {
		return strconv.FormatInt(allocated, 10)
	}
	defer tx.Rollback()

	err = tx.QueryRow(`SELECT next_value FROM id_sequences WHERE name = ? FOR UPDATE`, name).Scan(&allocated)
	if err == sql.ErrNoRows {
		allocated = start
		if _, err := tx.Exec(`INSERT INTO id_sequences (name, next_value) VALUES (?, ?)`, name, start+1); err != nil {
			return strconv.FormatInt(start, 10)
		}
	} else if err != nil || allocated < start {
		return strconv.FormatInt(start, 10)
	} else if _, err := tx.Exec(`UPDATE id_sequences SET next_value = ? WHERE name = ?`, allocated+1, name); err != nil {
		return strconv.FormatInt(start, 10)
	}

	if err := tx.Commit(); err != nil {
		return strconv.FormatInt(start, 10)
	}
	return strconv.FormatInt(allocated, 10)
}
