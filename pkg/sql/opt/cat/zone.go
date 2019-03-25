// Copyright 2018 The Cockroach Authors.
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

package cat

import (
	"bytes"
	"fmt"

	"github.com/cockroachdb/cockroach/pkg/util/treeprinter"
)

// Zone is an interface to zone configuration information used by the optimizer.
// The optimizer prefers indexes with constraints that best match the locality
// of the gateway node that plans the query.
type Zone interface {
	// ReplicaConstraintsCount returns the number of replica constraint sets that
	// are part of this zone.
	ReplicaConstraintsCount() int

	// ReplicaConstraints returns the ith set of replica constraints in the zone,
	// where i < ReplicaConstraintsCount.
	ReplicaConstraints(i int) ReplicaConstraints
}

// ReplicaConstraints is a set of constraints that apply to one or more replicas
// of a range, restricting which nodes can host that range. For example, if a
// table range has three replicas, then two of the replicas might be pinned to
// nodes in one region, whereas the third might be pinned to another region.
type ReplicaConstraints interface {
	// ReplicaCount returns the number of replicas that should abide by this set
	// of constraints. If 0, then the constraints apply to all replicas of the
	// range (and there can be only one ReplicaConstraints in the Zone).
	ReplicaCount() int32

	// ConstraintCount returns the number of constraints in the set.
	ConstraintCount() int

	// Constraint returns the ith constraint in the set, where
	// i < ConstraintCount.
	Constraint(i int) Constraint
}

// Constraint governs placement of range replicas on nodes. A constraint can
// either be required or prohibited. A required constraint's key/value pair must
// match one of the tiers of a node's locality for the range to locate there.
// A prohibited constraint's key/value pair must *not* match any of the tiers of
// a node's locality for the range to locate there. For example:
//
//   +region=east     Range can only be placed on nodes in region=east locality.
//   -region=west     Range cannot be placed on nodes in region=west locality.
//
type Constraint interface {
	// IsRequired is true if this is a required constraint, or false if this is
	// a prohibited constraint (signified by initial + or - character).
	IsRequired() bool

	// GetKey returns the constraint's string key (to left of =).
	GetKey() string

	// GetValue returns the constraint's string value (to right of =).
	GetValue() string
}

// FormatZone nicely formats a catalog zone using a treeprinter for debugging
// and testing.
func FormatZone(zone Zone, tp treeprinter.Node) {
	child := tp.Childf("ZONE")
	if zone.ReplicaConstraintsCount() > 1 {
		child = child.Childf("replica constraints")
	}
	for i, n := 0, zone.ReplicaConstraintsCount(); i < n; i++ {
		replConstraint := zone.ReplicaConstraints(i)
		constraintStr := formatReplicaConstraint(replConstraint)
		if zone.ReplicaConstraintsCount() > 1 {
			numReplicas := replConstraint.ReplicaCount()
			child.Childf("%d replicas: %s", numReplicas, constraintStr)
		} else {
			child.Childf("constraints: %s", constraintStr)
		}
	}
}

func formatReplicaConstraint(replConstraint ReplicaConstraints) string {
	var buf bytes.Buffer
	buf.WriteRune('[')
	for i, n := 0, replConstraint.ConstraintCount(); i < n; i++ {
		constraint := replConstraint.Constraint(i)
		if i != 0 {
			buf.WriteRune(',')
		}
		if constraint.IsRequired() {
			buf.WriteRune('+')
		} else {
			buf.WriteRune('-')
		}
		if constraint.GetKey() != "" {
			fmt.Fprintf(&buf, "%s=%s", constraint.GetKey(), constraint.GetValue())
		} else {
			buf.WriteString(constraint.GetValue())
		}
	}
	buf.WriteRune(']')
	return buf.String()
}
