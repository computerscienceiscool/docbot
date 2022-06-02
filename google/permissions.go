package google

import "google.golang.org/api/drive/v2"

// permissions example:
// https://github.com/kayac/alphawing/blob/52f67ecb99394dd263e7e33b8f73394e939f53aa/app/models/googleservice.go#L181
func (gf *Folder) CreateUserPermission(email string, role string) *drive.Permission {
	// Role: The primary role for this user. While new values may be
	// supported in the future, the following are currently allowed:
	// - owner
	// - organizer
	// - fileOrganizer
	// - writer
	// - reader
	return &drive.Permission{
		Role:  role,
		Type:  "user",
		Value: email,
	}
}

func (gf *Folder) CreateAnyonePermission(role string) *drive.Permission {
	return &drive.Permission{
		Role:     role,
		Type:     "anyone",
		WithLink: true,
	}
}

func (gf *Folder) InsertPermission(fileId string, permission *drive.Permission) (*drive.Permission, error) {
	return gf.drive.Permissions.Insert(fileId, permission).Do()
}

func (gf *Folder) GetPermissionList(fileId string) (*drive.PermissionList, error) {
	return gf.drive.Permissions.List(fileId).Do()
}

func (gf *Folder) UpdatePermission(fileId string, permissionId string, permission *drive.Permission) (*drive.Permission, error) {
	return gf.drive.Permissions.Update(fileId, permissionId, permission).Do()
}

func (gf *Folder) DeletePermission(fileId string, permissionId string) error {
	return gf.drive.Permissions.Delete(fileId, permissionId).Do()
}
