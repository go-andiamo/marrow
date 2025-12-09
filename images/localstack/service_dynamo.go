package localstack

import (
	"context"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/go-andiamo/marrow"
	"github.com/go-andiamo/marrow/framing"
	"github.com/go-andiamo/marrow/with"
	"strconv"
	"strings"
)

type DynamoService interface {
	Client() *dynamodb.Client
	PutItem(tableName string, item map[string]any) error
	GetItem(tableName string, keyProperty string, keyValue any) (map[string]any, error)
	DeleteItem(tableName string, keyProperty string, keyValue any) error
	CountItems(tableName string) (int64, error)
}

type dynamoImage struct {
	options    Options
	host       string
	mappedPort string
	client     *dynamodb.Client
}

var _ with.Image = (*dynamoImage)(nil)
var _ with.ImageResolveEnv = (*dynamoImage)(nil)
var _ DynamoService = (*dynamoImage)(nil)

func (i *image) createDynamoImage(ctx context.Context, awsCfg aws.Config) (err error) {
	img := &dynamoImage{
		options:    i.options,
		host:       i.host,
		mappedPort: i.mappedPort,
		client: dynamodb.NewFromConfig(awsCfg,
			func(o *dynamodb.Options) {
				o.BaseEndpoint = i.baseEndpoint()
				o.EndpointResolverV2 = dynamodb.NewDefaultEndpointResolverV2()
			},
		),
	}
	err = img.createTables(ctx)
	if err == nil {
		i.services[Dynamo] = img
	}
	return err
}

func (s *dynamoImage) createTables(ctx context.Context) error {
	for _, ct := range s.options.Dynamo.CreateTables {
		if ct.BillingMode == "" && ct.ProvisionedThroughput == nil {
			ct.BillingMode = types.BillingModePayPerRequest
		}
		if _, err := s.client.CreateTable(ctx, &ct); err != nil {
			return err
		}
	}
	return nil
}

func (s *dynamoImage) Client() *dynamodb.Client {
	return s.client
}

func (s *dynamoImage) PutItem(tableName string, item map[string]any) (err error) {
	var avs map[string]types.AttributeValue
	if avs, err = attributevalue.MarshalMap(item); err == nil {
		_, err = s.client.PutItem(context.Background(), &dynamodb.PutItemInput{
			TableName: &tableName,
			Item:      avs,
		})
	}
	return err
}

func (s *dynamoImage) GetItem(tableName string, keyProperty string, keyValue any) (result map[string]any, err error) {
	var kv types.AttributeValue
	if kv, err = valueToAttributeValue(keyValue); err == nil {
		var out *dynamodb.GetItemOutput
		if out, err = s.client.GetItem(context.Background(), &dynamodb.GetItemInput{
			TableName: &tableName,
			Key: map[string]types.AttributeValue{
				keyProperty: kv,
			},
			ConsistentRead: aws.Bool(true),
		}); err == nil {
			err = attributevalue.UnmarshalMap(out.Item, &result)
		}
	}
	return result, err
}

func (s *dynamoImage) DeleteItem(tableName string, keyProperty string, keyValue any) (err error) {
	var kv types.AttributeValue
	if kv, err = valueToAttributeValue(keyValue); err == nil {
		_, err = s.client.DeleteItem(context.Background(), &dynamodb.DeleteItemInput{
			TableName: &tableName,
			Key: map[string]types.AttributeValue{
				keyProperty: kv,
			},
		})
	}
	return err
}

func (s *dynamoImage) CountItems(tableName string) (int64, error) {
	var total int64
	var esk map[string]types.AttributeValue
	for {
		out, err := s.client.Scan(context.Background(), &dynamodb.ScanInput{
			TableName:         &tableName,
			Select:            types.SelectCount,
			ExclusiveStartKey: esk,
			ConsistentRead:    aws.Bool(true),
		})
		if err != nil {
			return 0, err
		}
		total += int64(out.Count)
		if len(out.LastEvaluatedKey) == 0 {
			break
		}
		esk = out.LastEvaluatedKey
	}
	return total, nil
}

func valueToAttributeValue(v any) (types.AttributeValue, error) {
	switch vt := v.(type) {
	case nil:
		return &types.AttributeValueMemberNULL{Value: true}, nil
	case types.AttributeValue:
		return vt, nil
	case []types.AttributeValue:
		return &types.AttributeValueMemberL{Value: vt}, nil
	case string:
		return &types.AttributeValueMemberS{Value: vt}, nil
	case []string:
		return &types.AttributeValueMemberSS{Value: vt}, nil
	case bool:
		return &types.AttributeValueMemberBOOL{Value: vt}, nil
	case []byte:
		return &types.AttributeValueMemberB{Value: vt}, nil
	case [][]byte:
		return &types.AttributeValueMemberBS{Value: vt}, nil
	case float32:
		return &types.AttributeValueMemberN{Value: strconv.FormatFloat(float64(vt), 'f', -1, 32)}, nil
	case float64:
		return &types.AttributeValueMemberN{Value: strconv.FormatFloat(vt, 'f', -1, 64)}, nil
	case int:
		return &types.AttributeValueMemberN{Value: strconv.Itoa(vt)}, nil
	case int64:
		return &types.AttributeValueMemberN{Value: strconv.Itoa(int(vt))}, nil
	case map[string]any:
		avm := make(map[string]types.AttributeValue, len(vt))
		for k, mv := range vt {
			if av, err := valueToAttributeValue(mv); err == nil {
				avm[k] = av
			} else {
				return nil, err
			}
		}
		return &types.AttributeValueMemberM{Value: avm}, nil
	}
	return nil, fmt.Errorf("unknown DynamoDB attribute type: %T", v)
}

const DynamoImageName = "dynamo"

func (s *dynamoImage) Name() string {
	return DynamoImageName
}

func (s *dynamoImage) Host() string {
	return s.host
}

func (s *dynamoImage) Port() string {
	return defaultPort
}

func (s *dynamoImage) MappedPort() string {
	return s.mappedPort
}

func (s *dynamoImage) IsDocker() bool {
	return true
}

func (s *dynamoImage) Username() string {
	return ""
}

func (s *dynamoImage) Password() string {
	return ""
}

func (s *dynamoImage) ResolveEnv(tokens ...string) (string, bool) {
	if len(tokens) > 0 {
		switch strings.ToLower(tokens[0]) {
		case "region":
			return s.options.region(), true
		case "accesskey":
			return s.options.accessKey(), true
		case "secretkey":
			return s.options.secretKey(), true
		case "sessiontoken":
			return s.options.sessionToken(), true
		}
	}
	return "", false
}

// DynamoPutItem can be used as a before/after on marrow.Method .Capture
// and puts an item into a Dynamo table
//
//go:noinline
func DynamoPutItem(when marrow.When, tableName string, item any, imgName ...string) marrow.BeforeAfter {
	return &capture[DynamoService]{
		name:     fmt.Sprintf("DynamoPutItem(%q)", tableName),
		when:     when,
		defImage: DynamoImageName,
		imgName:  imgName,
		run: func(ctx marrow.Context, img DynamoService) (err error) {
			var actual any
			if actual, err = marrow.ResolveValue(item, ctx); err == nil {
				if am, ok := actual.(map[string]any); ok {
					err = img.PutItem(tableName, am)
				} else {
					err = fmt.Errorf("unknown DynamoDB item type: %T", actual)
				}
			}
			return err
		},
		frame: framing.NewFrame(0),
	}
}

// DynamoDeleteItem can be used as a before/after on marrow.Method .Capture
// and deletes an item from a Dynamo table
//
//go:noinline
func DynamoDeleteItem(when marrow.When, tableName string, keyProperty string, keyValue any, imgName ...string) marrow.BeforeAfter {
	return &capture[DynamoService]{
		name:     fmt.Sprintf("DynamoDeleteItem(%q)", tableName),
		when:     when,
		defImage: DynamoImageName,
		imgName:  imgName,
		run: func(ctx marrow.Context, img DynamoService) (err error) {
			var akv any
			if akv, err = marrow.ResolveValue(keyValue, ctx); err == nil {
				err = img.DeleteItem(tableName, keyProperty, akv)
			}
			return err
		},
		frame: framing.NewFrame(0),
	}
}

// DynamoGetItem can be used as a resolvable value (e.g. in marrow.Method .AssertEqual)
// and resolves to the item in a dynamo table
//
//go:noinline
func DynamoGetItem(tableName string, keyProperty string, keyValue any, imgName ...string) marrow.Resolvable {
	return &resolvable[DynamoService]{
		name:     fmt.Sprintf("DynamoGetItem(%q)", tableName),
		defImage: DynamoImageName,
		imgName:  imgName,
		run: func(ctx marrow.Context, img DynamoService) (result any, err error) {
			var akv any
			if akv, err = marrow.ResolveValue(keyValue, ctx); err == nil {
				result, err = img.GetItem(tableName, keyProperty, akv)
			}
			return result, err
		},
		frame: framing.NewFrame(0),
	}
}

// DynamoItemsCount can be used as a resolvable value (e.g. in marrow.Method .AssertEqual)
// and resolves to the count of items in a dynamo table
//
//go:noinline
func DynamoItemsCount(tableName string, imgName ...string) marrow.Resolvable {
	return &resolvable[DynamoService]{
		name:     fmt.Sprintf("DynamoItemsCount(%q)", tableName),
		defImage: DynamoImageName,
		imgName:  imgName,
		run: func(ctx marrow.Context, img DynamoService) (result any, err error) {
			return img.CountItems(tableName)
		},
		frame: framing.NewFrame(0),
	}
}
