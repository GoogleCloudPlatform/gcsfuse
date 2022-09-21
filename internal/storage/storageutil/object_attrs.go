package storageutil

import (
	"crypto/md5"
	"time"

	"cloud.google.com/go/storage"
	"github.com/jacobsa/gcloud/gcs"
	storagev1 "google.golang.org/api/storage/v1"
)

func ObjectAttrsToBucketObject(attrs *storage.ObjectAttrs) *gcs.Object {
	// Converting []ACLRule returned by the Go Client into []*storagev1.ObjectAccessControl which complies with GCSFuse type.
	var acl []*storagev1.ObjectAccessControl
	for _, element := range attrs.ACL {
		currACL := &storagev1.ObjectAccessControl{
			Entity:   string(element.Entity),
			EntityId: element.EntityID,
			Role:     string(element.Role),
			Domain:   element.Domain,
			Email:    element.Email,
			ProjectTeam: &storagev1.ObjectAccessControlProjectTeam{
				ProjectNumber: element.ProjectTeam.ProjectNumber,
				Team:          element.ProjectTeam.Team,
			},
		}
		acl = append(acl, currACL)
	}

	// Converting MD5[] slice to MD5[md5.Size] type fixed array as accepted by GCSFuse.
	var MD5 [md5.Size]byte
	copy(MD5[:], attrs.MD5)

	// Setting the parameters in Object and doing conversions as necessary.
	return &gcs.Object{
		Name:            attrs.Name,
		ContentType:     attrs.ContentType,
		ContentLanguage: attrs.ContentLanguage,
		CacheControl:    attrs.CacheControl,
		Owner:           attrs.Owner,
		Size:            uint64(attrs.Size),
		ContentEncoding: attrs.ContentEncoding,
		MD5:             &MD5,
		CRC32C:          &attrs.CRC32C,
		MediaLink:       attrs.MediaLink,
		Metadata:        attrs.Metadata,
		Generation:      attrs.Generation,
		MetaGeneration:  attrs.Metageneration,
		StorageClass:    attrs.StorageClass,
		Deleted:         attrs.Deleted,
		Updated:         attrs.Updated,
		//ComponentCount: , (Field not found in attrs returned by Go Client.)
		ContentDisposition: attrs.ContentDisposition,
		CustomTime:         string(attrs.CustomTime.Format(time.RFC3339)),
		EventBasedHold:     attrs.EventBasedHold,
		Acl:                acl,
	}
}
