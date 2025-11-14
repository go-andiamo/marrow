package mongo

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-andiamo/marrow"
	"github.com/go-andiamo/marrow/framing"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

// DatabasesCount can be used as a resolvable value (e.g. in marrow.Method .AssertEqual)
// and resolves to the number of databases in a MongoDB
//
//go:noinline
func DatabasesCount(imgName ...string) marrow.Resolvable {
	return &resolvable{
		name:    "DatabasesCount()",
		imgName: imgName,
		run: func(ctx marrow.Context, img *image) (result any, err error) {
			result = 0
			var names []string
			if names, err = img.client.ListDatabaseNames(context.Background(), bson.D{}); err == nil {
				result = len(names)
			}
			return result, err
		},
		frame: framing.NewFrame(0),
	}
}

// CollectionsCount can be used as a resolvable value (e.g. in marrow.Method .AssertEqual)
// and resolves to the number of collections in a named MongoDB database
//
//go:noinline
func CollectionsCount(dbName string, imgName ...string) marrow.Resolvable {
	return &resolvable{
		name:    fmt.Sprintf("CollectionsCount(%q)", dbName),
		imgName: imgName,
		run: func(ctx marrow.Context, img *image) (result any, err error) {
			result = 0
			db := img.client.Database(dbName)
			var names []string
			if names, err = db.ListCollectionNames(context.Background(), bson.D{}); err == nil {
				result = len(names)
			}
			return result, err
		},
		frame: framing.NewFrame(0),
	}
}

// IndicesCount can be used as a resolvable value (e.g. in marrow.Method .AssertEqual)
// and resolves to the number of indices in a named MongoDB database and collection
//
//go:noinline
func IndicesCount(dbName string, collName string, imgName ...string) marrow.Resolvable {
	return &resolvable{
		name:    fmt.Sprintf("IndicesCount(%q, %q)", dbName, collName),
		imgName: imgName,
		run: func(ctx marrow.Context, img *image) (result any, err error) {
			result = 0
			coll := img.client.Database(dbName).Collection(collName)
			var idxs []mongo.IndexSpecification
			if idxs, err = coll.Indexes().ListSpecifications(context.Background()); err == nil {
				result = len(idxs)
			}
			return result, err
		},
		frame: framing.NewFrame(0),
	}
}

// DocumentsCount can be used as a resolvable value (e.g. in marrow.Method .AssertEqual)
// and resolves to the number of documents in a named MongoDB database and collection
//
//go:noinline
func DocumentsCount(dbName string, collName string, imgName ...string) marrow.Resolvable {
	return &resolvable{
		name:    fmt.Sprintf("DocumentsCount(%q, %q)", dbName, collName),
		imgName: imgName,
		run: func(ctx marrow.Context, img *image) (result any, err error) {
			return img.client.Database(dbName).Collection(collName).CountDocuments(context.Background(), bson.M{})
		},
		frame: framing.NewFrame(0),
	}
}

// FindOne can be used as a resolvable value (e.g. in marrow.Method .AssertEqual)
// and resolves to the document in the named MongoDB database and collection
//
//go:noinline
func FindOne(dbName string, collName string, filter any, opts *options.FindOneOptionsBuilder, imgName ...string) marrow.Resolvable {
	return &resolvable{
		name:    fmt.Sprintf("FindOne(%q, %q)", dbName, collName),
		imgName: imgName,
		run: func(ctx marrow.Context, img *image) (result any, err error) {
			var af any
			if af, err = marrow.ResolveValue(filter, ctx); err == nil {
				if af == nil {
					af = bson.D{}
				}
				mr := map[string]any{}
				sr := img.client.Database(dbName).Collection(collName).FindOne(context.Background(), af, opts)
				if err = sr.Decode(&mr); err == nil {
					result = mr
				} else if errors.Is(err, mongo.ErrNoDocuments) {
					err = nil
				}
			}
			return result, err
		},
		frame: framing.NewFrame(0),
	}
}

// Find can be used as a resolvable value (e.g. in marrow.Method .AssertEqual)
// and resolves to the documents in the named MongoDB database and collection
//
//go:noinline
func Find(dbName string, collName string, filter any, opts *options.FindOptionsBuilder, imgName ...string) marrow.Resolvable {
	return &resolvable{
		name:    fmt.Sprintf("Find(%q, %q)", dbName, collName),
		imgName: imgName,
		run: func(ctx marrow.Context, img *image) (result any, err error) {
			var af any
			if af, err = marrow.ResolveValue(filter, ctx); err == nil {
				if af == nil {
					af = bson.D{}
				}
				var csr *mongo.Cursor
				if csr, err = img.client.Database(dbName).Collection(collName).Find(context.Background(), af, opts); err == nil {
					lr := []map[string]any{}
					if err = csr.All(context.Background(), &lr); err == nil {
						result = lr
					}
				}
			}
			return result, err
		},
		frame: framing.NewFrame(0),
	}
}

// InsertDocument can be used as a before/after on marrow.Method .Capture
// and inserts a document into the named MongoDB database and collection
//
//go:noinline
func InsertDocument(when marrow.When, dbName string, collName string, doc any, imgName ...string) marrow.BeforeAfter {
	return &capture{
		when:    when,
		name:    fmt.Sprintf("InsertDocument(%q, %q)", dbName, collName),
		imgName: imgName,
		run: func(ctx marrow.Context, img *image) (err error) {
			var av any
			if av, err = marrow.ResolveValue(doc, ctx); err == nil {
				_, err = img.client.Database(dbName).Collection(collName).InsertOne(context.Background(), av)
			}
			return err
		},
		frame: framing.NewFrame(0),
	}
}

// ClearCollection can be used as a before/after on marrow.Method .Capture
// and clears (deletss) all documents in the named MongoDB database and collection
//
//go:noinline
func ClearCollection(when marrow.When, dbName string, collName string, imgName ...string) marrow.BeforeAfter {
	return &capture{
		when:    when,
		name:    fmt.Sprintf("ClearCollection(%q, %q)", dbName, collName),
		imgName: imgName,
		run: func(ctx marrow.Context, img *image) (err error) {
			_, err = img.client.Database(dbName).Collection(collName).DeleteMany(context.Background(), bson.M{})
			return err
		},
		frame: framing.NewFrame(0),
	}
}

// DeleteDocuments can be used as a before/after on marrow.Method .Capture
// and deletes all documents matching the provided filter in the named MongoDB database and collection
//
//go:noinline
func DeleteDocuments(when marrow.When, dbName string, collName string, filter any, imgName ...string) marrow.BeforeAfter {
	return &capture{
		when:    when,
		name:    fmt.Sprintf("DeleteDocuments(%q, %q)", dbName, collName),
		imgName: imgName,
		run: func(ctx marrow.Context, img *image) (err error) {
			var af any
			if af, err = marrow.ResolveValue(filter, ctx); err == nil {
				if af == nil {
					af = bson.D{}
				}
				_, err = img.client.Database(dbName).Collection(collName).DeleteMany(context.Background(), af)
			}
			return err
		},
		frame: framing.NewFrame(0),
	}
}

// Query can be used as a resolvable value (e.g. in marrow.Method .AssertEqual)
// and resolves to the documents retrieved by the supplied query in the named MongoDB database
//
// the query arg is resolved and can be a string, map, etc. - strings are unmarshalled as bson
//
//go:noinline
func Query(dbName string, query any, imgName ...string) marrow.Resolvable {
	return &resolvable{
		name:    fmt.Sprintf("Query(%q)", dbName),
		imgName: imgName,
		run: func(ctx marrow.Context, img *image) (result any, err error) {
			var aq any
			if aq, err = marrow.ResolveValue(query, ctx); err == nil {
				cmd := aq
				switch aqt := aq.(type) {
				case nil:
					err = errors.New("query is nil")
				case string:
					err = bson.UnmarshalExtJSON([]byte(aqt), true, &cmd)
				}
				if err == nil {
					var csr *mongo.Cursor
					if csr, err = img.client.Database(dbName).RunCommandCursor(context.Background(), cmd); err == nil {
						lr := []map[string]any{}
						if err = csr.All(context.Background(), &lr); err == nil {
							result = lr
						}
					}
				}
			}
			return result, err
		},
		frame: framing.NewFrame(0),
	}
}

type capture struct {
	name    string
	when    marrow.When
	imgName []string
	run     func(ctx marrow.Context, img *image) error
	frame   *framing.Frame
}

var _ marrow.BeforeAfter = (*capture)(nil)

func (c *capture) When() marrow.When {
	return c.when
}

func (c *capture) Run(ctx marrow.Context) (err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error running operation %s: %w", c.name, err)
		}
	}()
	var img *image
	if img, err = imageFromContext(ctx, c.imgName); err == nil {
		err = c.run(ctx, img)
	}
	return err
}

func (c *capture) Frame() *framing.Frame {
	return c.frame
}

type resolvable struct {
	name    string
	imgName []string
	run     func(ctx marrow.Context, img *image) (any, error)
	frame   *framing.Frame
}

var _ marrow.Resolvable = (*resolvable)(nil)
var _ fmt.Stringer = (*resolvable)(nil)

func (r *resolvable) ResolveValue(ctx marrow.Context) (av any, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("error resolving value %s: %w", r.name, err)
		}
	}()
	var img *image
	if img, err = imageFromContext(ctx, r.imgName); err == nil {
		av, err = r.run(ctx, img)
	}
	return av, err
}

func (r *resolvable) String() string {
	return imageName + "." + r.name
}

func imageFromContext(ctx marrow.Context, name []string) (*image, error) {
	n := imageName
	if len(name) > 0 && name[0] != "" {
		n = name[0]
	}
	if i := ctx.GetImage(n); i != nil {
		if img, ok := i.(*image); ok {
			return img, nil
		}
	}
	return nil, fmt.Errorf("image not found: %s", name)
}
