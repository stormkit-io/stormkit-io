package volumes

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"text/template"

	"github.com/lib/pq"
	"github.com/stormkit-io/stormkit-io/src/lib/database"
	"github.com/stormkit-io/stormkit-io/src/lib/types"
	"github.com/stormkit-io/stormkit-io/src/lib/utils"
)

var stmt = struct {
	insertFiles      string
	selectFiles      string
	removeFiles      string
	volumeSize       string
	changeVisibility string
}{
	selectFiles: `
		SELECT
			file_id, env_id, file_name, file_path,
			file_size, file_metadata,
			is_public, created_at, updated_at
		FROM
			volumes
		WHERE
			{{ .where }}
		ORDER BY
			{{ .orderBy }} DESC
		LIMIT
			{{ .limit }};
	`,

	removeFiles: `
		DELETE FROM volumes WHERE file_id = ANY($1) AND env_id = $2;
	`,

	insertFiles: `
		INSERT INTO volumes (
			file_name,
			file_path,
			file_size,
			is_public,
			env_id,
			created_at
		)
		VALUES
			{{ generateValues 6 (len .) }}
		ON CONFLICT
			(file_name, env_id)
		DO UPDATE SET
			updated_at = EXCLUDED.created_at
		RETURNING
			file_id
	`,

	volumeSize: `
		SELECT SUM(file_size) FROM volumes WHERE env_id = $1;
	`,

	changeVisibility: `
		UPDATE volumes SET is_public = $1 WHERE file_id = $2;
	`,
}

// Store represents a store for volume management.
type store struct {
	*database.Store
	insertTmpl *template.Template
	selectTmpl *template.Template
}

// Store returns a new store instance.
func Store() *store {
	return &store{
		Store: database.NewStore(),
		selectTmpl: template.Must(
			template.New("selectFiles").
				Parse(stmt.selectFiles),
		),
		insertTmpl: template.Must(
			template.New("insertFiles").
				Funcs(template.FuncMap{"generateValues": utils.GenerateValues}).
				Parse(stmt.insertFiles)),
	}
}

type SelectFilesArgs struct {
	OrderBy  string
	EnvID    types.ID
	BeforeID types.ID
	FileID   []types.ID
	Limit    int
}

// SelectFiles selects the files with the given arguments.
func (s *store) SelectFiles(ctx context.Context, args SelectFilesArgs) ([]*File, error) {
	var qb strings.Builder
	orderBy := "file_id"
	params := []any{}
	where := []string{}
	limit := 100

	if args.OrderBy == "name" {
		orderBy = "file_name"
	}

	if args.EnvID > 0 {
		where = append(where, "env_id = $1")
		params = append(params, args.EnvID)
	}

	if args.BeforeID > 0 {
		where = append(where, fmt.Sprintf("file_id < $%d", len(params)+1))
		params = append(params, args.BeforeID)
	}

	if len(args.FileID) > 0 {
		where = append(where, fmt.Sprintf("file_id = ANY($%d)", len(params)+1))
		params = append(params, pq.Array(args.FileID))
	}

	if args.Limit > 0 {
		limit = args.Limit
	}

	data := map[string]any{
		"orderBy": orderBy,
		"where":   strings.Join(where, " AND "),
		"limit":   limit,
	}

	if err := s.selectTmpl.Execute(&qb, data); err != nil {
		return nil, err
	}

	rows, err := s.Query(ctx, qb.String(), params...)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	defer rows.Close()

	files := []*File{}

	for rows.Next() {
		file := &File{}
		err := rows.Scan(
			&file.ID, &file.EnvID, &file.Name, &file.Path, &file.Size,
			&file.Metadata, &file.IsPublic, &file.CreatedAt, &file.UpdatedAt,
		)

		if err != nil {
			return nil, err
		}

		if file.Metadata == nil || file.Metadata["mountType"] == "" {
			file.Metadata = utils.Map{"mountType": FileSys}
		}

		files = append(files, file)
	}

	return files, nil
}

// FileByID returns the given file by it's ID.
func (s *store) FileByID(ctx context.Context, fileID types.ID) (*File, error) {
	files, err := s.SelectFiles(ctx, SelectFilesArgs{Limit: 1, FileID: []types.ID{fileID}})

	if err != nil {
		return nil, err
	}

	if len(files) > 0 {
		return files[0], nil
	}

	return nil, nil
}

// Insert the given batch into the database.
func (s *store) Insert(ctx context.Context, files []*File, envID types.ID) error {
	var qb strings.Builder

	if err := s.insertTmpl.Execute(&qb, files); err != nil {
		return err
	}

	params := []any{}

	for _, file := range files {
		params = append(params,
			file.Name, file.Path, file.Size, file.IsPublic, envID, file.CreatedAt,
		)
	}

	rows, err := s.Query(ctx, qb.String(), params...)

	if err != nil {
		return err
	}

	if rows == nil {
		return nil
	}

	defer rows.Close()

	i := 0

	for rows.Next() {
		if err := rows.Scan(&files[i].ID); err != nil {
			return err
		}

		i = i + 1
	}

	return err
}

// RemoveFiles removes files from the database.
func (s *store) RemoveFiles(ctx context.Context, files []*File, envID types.ID) error {
	fileIDs := []types.ID{}

	for _, file := range files {
		fileIDs = append(fileIDs, file.ID)
	}

	_, err := s.Exec(ctx, stmt.removeFiles, pq.Array(fileIDs), envID)
	return err
}

// VolumeSize returns the volume size for the given environment.
func (s *store) VolumeSize(ctx context.Context, envID types.ID) (int64, error) {
	row, err := s.QueryRow(ctx, stmt.volumeSize, envID)

	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}

		return 0, err
	}

	var size int64

	if err := row.Scan(&size); err != nil {
		return 0, err
	}

	return size, nil
}

// ChangeVisibility changes the visibility of the given file.
func (s *store) ChangeVisibility(ctx context.Context, fileID types.ID, isPublic bool) error {
	_, err := s.Exec(ctx, stmt.changeVisibility, isPublic, fileID)
	return err
}
