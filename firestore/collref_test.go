// Copyright 2017 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package firestore

import (
	"context"
	"testing"
	"time"

	"github.com/golang/protobuf/proto"
	pb "google.golang.org/genproto/googleapis/firestore/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestDoc(t *testing.T) {
	coll := testClient.Collection("C")
	got := coll.Doc("d")
	want := &DocumentRef{
		Parent:       coll,
		ID:           "d",
		Path:         "projects/projectID/databases/(default)/documents/C/d",
		shortPath:    "C/d",
		readSettings: &readSettings{},
	}
	if !testEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestNewDoc(t *testing.T) {
	c := &Client{}
	coll := c.Collection("C")
	got := coll.NewDoc()
	if got.Parent != coll {
		t.Errorf("NewDoc got %v, want %v", got.Parent, coll)
	}
	if len(got.ID) != 20 {
		t.Errorf("got %d-char ID, wanted 20", len(got.ID))
	}

	got2 := coll.NewDoc()
	if got.ID == got2.ID {
		t.Error("got same ID")
	}
}

func TestAdd(t *testing.T) {
	ctx := context.Background()
	c, srv, cleanup := newMock(t)
	defer cleanup()

	wantReq := commitRequestForSet()
	w := wantReq.Writes[0]
	w.CurrentDocument = &pb.Precondition{
		ConditionType: &pb.Precondition_Exists{false},
	}
	srv.addRPCAdjust(wantReq, commitResponseForSet, func(gotReq proto.Message) {
		// We can't know the doc ID before Add is called, so we take it from
		// the request.
		w.Operation.(*pb.Write_Update).Update.Name = gotReq.(*pb.CommitRequest).Writes[0].Operation.(*pb.Write_Update).Update.Name
	})
	_, wr, err := c.Collection("C").Add(ctx, testData)
	if err != nil {
		t.Fatal(err)
	}
	if !testEqual(wr, writeResultForSet) {
		t.Errorf("got %v, want %v", wr, writeResultForSet)
	}
}

func TestNilErrors(t *testing.T) {
	ctx := context.Background()
	c, _, cleanup := newMock(t)
	defer cleanup()

	// Test that a nil CollectionRef results in a nil DocumentRef and errors
	// where possible.
	coll := c.Collection("a/b") // nil because "a/b" denotes a doc.
	if coll != nil {
		t.Fatal("collection not nil")
	}
	if got := coll.Doc("d"); got != nil {
		t.Fatalf("got %v, want nil", got)
	}
	if got := coll.NewDoc(); got != nil {
		t.Fatalf("got %v, want nil", got)
	}
	if _, _, err := coll.Add(ctx, testData); err != errNilDocRef {
		t.Fatalf("got <%v>, want <%v>", err, errNilDocRef)
	}
}

func TestCollRef_WithReadOptions(t *testing.T) {
	ctx := context.Background()
	c, srv, cleanup := newMock(t)
	defer cleanup()

	const dbPath = "projects/projectID/databases/(default)"
	const docPath = dbPath + "/documents/C/a"
	tm := time.Date(2021, time.February, 20, 0, 0, 0, 0, time.UTC)

	srv.addRPC(&pb.ListDocumentsRequest{
		Parent:       dbPath,
		CollectionId: "myCollection",
		ShowMissing:  true,
		ConsistencySelector: &pb.ListDocumentsRequest_ReadTime{
			ReadTime: &timestamppb.Timestamp{Seconds: tm.Unix()},
		},
	}, []interface{}{
		&pb.ListDocumentsResponse{
			Documents: []*pb.Document{
				{
					Name: docPath,
				},
			},
		},
	})

	_, err := c.Collection("myCollection").WithReadOptions(ReadTime(tm)).DocumentRefs(ctx).GetAll()
	if err == nil {
		t.Fatal(err)
	}
}
