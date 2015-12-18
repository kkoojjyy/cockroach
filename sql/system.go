// Copyright 2015 The Cockroach Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or
// implied. See the License for the specific language governing
// permissions and limitations under the License.
//
// Author: Marc Berhault (marc@cockroachlabs.com)

package sql

import (
	"github.com/cockroachdb/cockroach/keys"
	"github.com/cockroachdb/cockroach/security"
	"github.com/cockroachdb/cockroach/sql/parser"
	"github.com/cockroachdb/cockroach/sql/privilege"
	"github.com/cockroachdb/cockroach/util/log"
)

const (
	// sql CREATE commands and full schema for each system table.
	namespaceTableSchema = `
CREATE TABLE system.namespace (
  parentID INT,
  name     STRING,
  id       INT,
  PRIMARY KEY (parentID, name)
);`

	descriptorTableSchema = `
CREATE TABLE system.descriptor (
  id         INT PRIMARY KEY,
  descriptor BYTES
);`

	leaseTableSchema = `
CREATE TABLE system.lease (
  descID     INT,
  version    INT,
  nodeID     INT,
  expiration TIMESTAMP,
  PRIMARY KEY (descID, version, expiration, nodeID)
);`

	usersTableSchema = `
CREATE TABLE system.users (
  username       STRING PRIMARY KEY,
  hashedPassword BYTES
);`

	// Zone settings per DB/Table.
	zonesTableSchema = `
CREATE TABLE system.zones (
  id     INT PRIMARY KEY,
  config BYTES
);`
)

var (
	// SystemDB is the descriptor for the system database.
	SystemDB = DatabaseDescriptor{
		Name: "system",
		ID:   keys.SystemDatabaseID,
		// Assign max privileges to root user.
		Privileges: NewPrivilegeDescriptor(security.RootUser,
			SystemAllowedPrivileges[keys.SystemDatabaseID]),
	}

	// NamespaceTable is the descriptor for the namespace table.
	NamespaceTable = createSystemTable(keys.NamespaceTableID, namespaceTableSchema)

	// DescriptorTable is the descriptor for the descriptor table.
	DescriptorTable = createSystemTable(keys.DescriptorTableID, descriptorTableSchema)

	// LeaseTable is the descriptor for the lease table.
	LeaseTable = createSystemTable(keys.LeaseTableID, leaseTableSchema)

	// UsersTable is the descriptor for the users table.
	UsersTable = createSystemTable(keys.UsersTableID, usersTableSchema)

	// ZonesTable is the descriptor for the zones table.
	ZonesTable = createSystemTable(keys.ZonesTableID, zonesTableSchema)

	// SystemAllowedPrivileges describes the privileges allowed for each
	// system object. No user may have more than those privileges, and
	// the root user must have exactly those privileges.
	// CREATE|DROP|ALL should always be denied.
	SystemAllowedPrivileges = map[ID]privilege.List{
		keys.SystemDatabaseID:  privilege.ReadData,
		keys.NamespaceTableID:  privilege.ReadData,
		keys.DescriptorTableID: privilege.ReadData,
		keys.LeaseTableID:      privilege.ReadWriteData,
		keys.UsersTableID:      privilege.ReadWriteData,
		keys.ZonesTableID:      privilege.ReadWriteData,
	}

	// NumSystemDescriptors should be set to the number of system descriptors
	// above (SystemDB and each system table). This is used by tests which need
	// to know the number of system descriptors intended for installation; it starts at
	// 1 for the SystemDB descriptor created above, and is incremented by every
	// call to createSystemTable().
	NumSystemDescriptors = 1
)

func createSystemTable(id ID, schema string) TableDescriptor {
	NumSystemDescriptors++

	// System tables have the system database as a parent, with privileges from
	// the SystemAllowedPrivileges table assigned to the root user.
	return createTableDescriptor(id, keys.SystemDatabaseID, schema,
		NewPrivilegeDescriptor(security.RootUser, SystemAllowedPrivileges[id]))
}

func createTableDescriptor(id, parentID ID, schema string, privileges *PrivilegeDescriptor) TableDescriptor {
	stmt, err := parser.ParseOneTraditional(schema)
	if err != nil {
		log.Fatal(err)
	}

	desc, err := makeTableDesc(stmt.(*parser.CreateTable), parentID)
	if err != nil {
		log.Fatal(err)
	}

	desc.Privileges = privileges

	desc.ID = id
	if err := desc.AllocateIDs(); err != nil {
		log.Fatal(err)
	}

	return desc
}

// addSystemDatabaseToSchema populates the supplied MetadataSchema with the
// System database and its tables. The descriptors for these objects exist
// statically in this file, but a MetadataSchema can be used to persist these
// descriptors to the cockroach store.
func addSystemDatabaseToSchema(target *MetadataSchema) {
	// Add system database.
	target.AddSystemDescriptor(keys.RootNamespaceID, &SystemDB)

	// Add system tables.
	target.AddSystemDescriptor(keys.SystemDatabaseID, &NamespaceTable)
	target.AddSystemDescriptor(keys.SystemDatabaseID, &DescriptorTable)
	target.AddSystemDescriptor(keys.SystemDatabaseID, &LeaseTable)
	target.AddSystemDescriptor(keys.SystemDatabaseID, &UsersTable)
	target.AddSystemDescriptor(keys.SystemDatabaseID, &ZonesTable)
}

// IsSystemID returns true if this ID is reserved for system objects.
func IsSystemID(id ID) bool {
	return id > 0 && id <= keys.MaxSystemDescID
}
