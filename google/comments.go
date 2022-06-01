package google

/*
   gf.drive.Comments.Delete(fileId string, commentId string) : *CommentsDeleteCall
   gf.drive.Comments.Get(fileId string, commentId string) : *CommentsGetCall
   gf.drive.Comments.Insert(fileId string, comment *Comment) : *CommentsInsertCall
   gf.drive.Comments.List(fileId string) : *CommentsListCall

func (gf *Folder) CreateComment(email string, role string) *drive.Permission {
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

func (gf *Folder) CreateAnyonePermission(email string, role string) *drive.Permission {
	return &drive.Permission{
		Role: role,
		Type: "anyone",
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

*/
