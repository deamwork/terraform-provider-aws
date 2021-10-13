// Code generated by internal/generate/tags/main.go; DO NOT EDIT.
package resourcegroups

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/resourcegroups"
	tftags "github.com/hashicorp/terraform-provider-aws/internal/tags"
)

// ListTags lists resourcegroups service tags.
// The identifier is typically the Amazon Resource Name (ARN), although
// it may also be a different identifier depending on the service.
func ListTags(conn *resourcegroups.ResourceGroups, identifier string) (tftags.KeyValueTags, error) {
	input := &resourcegroups.GetTagsInput{
		Arn: aws.String(identifier),
	}

	output, err := conn.GetTags(input)

	if err != nil {
		return tftags.New(nil), err
	}

	return KeyValueTags(output.Tags), nil
}

// map[string]*string handling

// Tags returns resourcegroups service tags.
func Tags(tags tftags.KeyValueTags) map[string]*string {
	return aws.StringMap(tags.Map())
}

// KeyValueTags creates KeyValueTags from resourcegroups service tags.
func KeyValueTags(tags map[string]*string) tftags.KeyValueTags {
	return tftags.New(tags)
}

// UpdateTags updates resourcegroups service tags.
// The identifier is typically the Amazon Resource Name (ARN), although
// it may also be a different identifier depending on the service.
func UpdateTags(conn *resourcegroups.ResourceGroups, identifier string, oldTagsMap interface{}, newTagsMap interface{}) error {
	oldTags := tftags.New(oldTagsMap)
	newTags := tftags.New(newTagsMap)

	if removedTags := oldTags.Removed(newTags); len(removedTags) > 0 {
		input := &resourcegroups.UntagInput{
			Arn:  aws.String(identifier),
			Keys: aws.StringSlice(removedTags.IgnoreAWS().Keys()),
		}

		_, err := conn.Untag(input)

		if err != nil {
			return fmt.Errorf("error untagging resource (%s): %w", identifier, err)
		}
	}

	if updatedTags := oldTags.Updated(newTags); len(updatedTags) > 0 {
		input := &resourcegroups.TagInput{
			Arn:  aws.String(identifier),
			Tags: Tags(updatedTags.IgnoreAWS()),
		}

		_, err := conn.Tag(input)

		if err != nil {
			return fmt.Errorf("error tagging resource (%s): %w", identifier, err)
		}
	}

	return nil
}
