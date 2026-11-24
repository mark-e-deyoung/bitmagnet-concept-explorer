package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestBitmagnetV010NativeMediaIdentityProjection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/graphql" || r.Method != http.MethodPost {
			t.Errorf("unexpected native upstream call")
			w.WriteHeader(http.StatusNotFound)
			return
		}
		var body struct { Query string `json:"query"` }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
			return
		}
		for _, field := range []string{"contentSource", "contentId", "releaseYear"} {
			if !strings.Contains(body.Query, field) {
				t.Errorf("native v0.10 identity field missing from GraphQL projection: %s", field)
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":{"torrentContent":{"search":{"hasNextPage":false,"items":[{
			"title":"Synthetic Original Movie","contentType":"movie","contentSource":"TMDB","contentId":"51230",
			"seeders":4,"content":{"adult":false,"releaseYear":2021},
			"torrent":{"infoHash":"1111111111111111111111111111111111111111","name":"fictional"}
		}]}}}}`))
	}))
	defer server.Close()
	t.Setenv("BCE_BITMAGNET_GRAPHQL", server.URL+"/graphql")
	got, err := execute(context.Background(), Request{Query:"Synthetic Original Movie"})
	if err != nil || got.State != "complete" || len(got.Results) != 1 {
		t.Fatalf("native identity result invalid %#v / %v", got, err)
	}
	v := got.Results[0]
	if v.ContentSource != "tmdb" || v.ContentID != "51230" ||
		v.ReleaseYear == nil || *v.ReleaseYear != 2021 {
		t.Fatalf("lost explicit provider identity and release year %#v", v)
	}
	if v.Title != "Synthetic Original Movie" || v.ContentType != "movie" ||
		v.AdultMetadata == nil || *v.AdultMetadata {
		t.Fatalf("corrupted existing observation fields %#v", v)
	}
	if got.ReadOnly != true || got.ExternalResults == nil && len(got.Failures)>0 {
		t.Fatalf("identity read changed source authority %#v", got)
	}
}

func TestMediaIdentityConflictsLatchUnknownWithoutArrivalOrder(t *testing.T) {
	year2020, year2024 := 2020, 2024
	base := Observation{
		InfoHash:"1111111111111111111111111111111111111111",
		Title:"Fictional Original", ContentType:"movie",
		ContentSource:"tmdb", ContentID:"100", ReleaseYear:&year2020,
	}
	other := Observation{
		InfoHash:base.InfoHash,Title:base.Title,ContentType:"movie",
		ContentSource:"tmdb",ContentID:"200",ReleaseYear:&year2024,
	}
	mergeObservationMetadata(&base,other)
	if base.ContentSource!="tmdb" || base.ContentID!="" || base.ReleaseYear!=nil ||
		!contains(base.ConflictingMetadata,"content_id") ||
		!contains(base.ConflictingMetadata,"release_year"){
		t.Fatalf("provider ID or year conflict wrongly resolved %#v",base)
	}
	mergeObservationMetadata(&base,Observation{
		InfoHash:base.InfoHash,ContentSource:"tmdb",ContentID:"100",ReleaseYear:&year2020,
	})
	if base.ContentID!="" || base.ReleaseYear!=nil {
		t.Fatal("later arrival unlatched identity conflict")
	}
	conflictingSource := Observation{
		InfoHash:base.InfoHash,ContentSource:"imdb",ContentID:"tt1234567",
	}
	mergeObservationMetadata(&base,conflictingSource)
	if base.ContentSource!="" || !contains(base.ConflictingMetadata,"content_source") {
		t.Fatalf("cross-provider conflict not signaled: %#v",base)
	}
}

func TestAbsentMediaIdentityDoesNotFabricateProviderID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"data":{"torrentContent":{"search":{"items":[{
			"title":"Synthetic Without Classification","contentType":"other",
			"content":{"adult":null,"releaseYear":2500},
			"torrent":{"infoHash":"2222222222222222222222222222222222222222"}
		}]}}}}`))
	}))
	defer server.Close()
	t.Setenv("BCE_BITMAGNET_GRAPHQL",server.URL+"/graphql")
	got,err:=execute(context.Background(),Request{Query:"Synthetic"})
	if err!=nil || got.State!="complete" || len(got.Results)!=1 {
		t.Fatalf("failed unclassified result %#v / %v",got,err)
	}
	item:=got.Results[0]
	if item.ContentSource!="" || item.ContentID!="" || item.ReleaseYear!=nil {
		t.Fatalf("fabricated unsupported provider identity %#v",item)
	}
}
