// Copyright 2022 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storageutil

import (
	"crypto/md5"
	"time"

	"cloud.google.com/go/storage"
	"github.com/jacobsa/gcloud/gcs"
	storagev1 "google.golang.org/api/storage/v1"
)

func convertObjectAccessControlToACLRule(obj *storagev1.ObjectAccessControl) storage.ACLRule {
	return storage.ACLRule{
		Entity:   storage.ACLEntity(obj.Entity),
		EntityID: obj.EntityId,
		Role:     storage.ACLRole(obj.Role),
		Domain:   obj.Domain,
		Email:    obj.Email,
		ProjectTeam: &storage.ProjectTeam{
			ProjectNumber: obj.ProjectTeam.ProjectNumber,
			Team:          obj.ProjectTeam.Team,
		},
	}
}

func convertACLRuleToObjectAccessControl(element storage.ACLRule) *storagev1.ObjectAccessControl {
	return &storagev1.ObjectAccessControl{
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
}

func ObjectAttrsToBucketObject(attrs *storage.ObjectAttrs) *gcs.Object {
	// gcs.Object accepts []*storagev1.ObjectAccessControl instead of []ACLRule.
	var acl []*storagev1.ObjectAccessControl
	for _, element := range attrs.ACL {
		acl = append(acl, convertACLRuleToObjectAccessControl(element))
	}

	// Converting MD5[] slice to MD5[md5.Size] type fixed array as accepted by GCSFuse.
	var md5 [md5.Size]byte
	copy(md5[:], attrs.MD5)

	// Making a local copy of crc to avoid keeping a reference to attrs instance.
	crc := attrs.CRC32C

	// Setting the parameters in Object and doing conversions as necessary.
	return &gcs.Object{
		Name:               attrs.Name,
		ContentType:        attrs.ContentType,
		ContentLanguage:    attrs.ContentLanguage,
		CacheControl:       attrs.CacheControl,
		Owner:              attrs.Owner,
		Size:               uint64(attrs.Size),
		ContentEncoding:    attrs.ContentEncoding,
		MD5:                &md5,
		CRC32C:             &crc,
		MediaLink:          attrs.MediaLink,
		Metadata:           attrs.Metadata,
		Generation:         attrs.Generation,
		MetaGeneration:     attrs.Metageneration,
		StorageClass:       attrs.StorageClass,
		Deleted:            attrs.Deleted,
		Updated:            attrs.Updated,
		ComponentCount:     attrs.ComponentCount,
		ContentDisposition: attrs.ContentDisposition,
		CustomTime:         string(attrs.CustomTime.Format(time.RFC3339)),
		EventBasedHold:     attrs.EventBasedHold,
		Acl:                acl,
	}
}

// SetAttrsInWriter - for setting object-attributes filed in storage.Writer object.
// These attributes will be assigned to the newly created or old object.
func SetAttrsInWriter(wc *storage.Writer, req *gcs.CreateObjectRequest) *storage.Writer {
	wc.Name = req.Name
	wc.ContentType = req.ContentType
	wc.ContentLanguage = req.ContentLanguage
	wc.ContentEncoding = req.ContentEncoding
	wc.CacheControl = req.CacheControl
	wc.Metadata = req.Metadata
	wc.ContentDisposition = req.ContentDisposition
	wc.CustomTime, _ = time.Parse(time.RFC3339, req.CustomTime)
	wc.EventBasedHold = req.EventBasedHold
	wc.StorageClass = req.StorageClass

	// Converting []*storagev1.ObjectAccessControl to []ACLRule for writer object.
	var aclRules []storage.ACLRule
	for _, element := range req.Acl {
		aclRules = append(aclRules, convertObjectAccessControlToACLRule(element))
	}
	wc.ACL = aclRules

	if req.CRC32C != nil {
		wc.CRC32C = *req.CRC32C
		wc.SendCRC32C = true
	}

	if req.MD5 != nil {
		wc.MD5 = (*req.MD5)[:]
	}

	return wc
}
