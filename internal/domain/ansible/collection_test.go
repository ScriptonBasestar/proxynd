package ansible

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateNamespace(t *testing.T) {
	tests := []struct {
		name      string
		namespace string
		wantErr   error
	}{
		{
			name:      "valid namespace",
			namespace: "community",
			wantErr:   nil,
		},
		{
			name:      "valid namespace with underscore",
			namespace: "my_company",
			wantErr:   nil,
		},
		{
			name:      "valid namespace with numbers",
			namespace: "company123",
			wantErr:   nil,
		},
		{
			name:      "empty namespace",
			namespace: "",
			wantErr:   ErrInvalidNamespace,
		},
		{
			name:      "namespace with uppercase",
			namespace: "MyCompany",
			wantErr:   ErrInvalidNamespace,
		},
		{
			name:      "namespace with hyphen",
			namespace: "my-company",
			wantErr:   ErrInvalidNamespace,
		},
		{
			name:      "namespace with special chars",
			namespace: "my@company",
			wantErr:   ErrInvalidNamespace,
		},
		{
			name:      "reserved namespace ansible",
			namespace: "ansible",
			wantErr:   ErrReservedNamespace,
		},
		{
			name:      "reserved namespace galaxy",
			namespace: "galaxy",
			wantErr:   ErrReservedNamespace,
		},
		{
			name:      "namespace too long",
			namespace: "this_is_a_very_long_namespace_name_that_exceeds_the_maximum_allowed_length_of_64_characters",
			wantErr:   ErrNamespaceTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNamespace(tt.namespace)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateName(t *testing.T) {
	tests := []struct {
		name    string
		colName string
		wantErr error
	}{
		{
			name:    "valid name",
			colName: "general",
			wantErr: nil,
		},
		{
			name:    "valid name with underscore",
			colName: "my_collection",
			wantErr: nil,
		},
		{
			name:    "valid name with numbers",
			colName: "collection123",
			wantErr: nil,
		},
		{
			name:    "empty name",
			colName: "",
			wantErr: ErrInvalidName,
		},
		{
			name:    "name with uppercase",
			colName: "MyCollection",
			wantErr: ErrInvalidName,
		},
		{
			name:    "name with hyphen",
			colName: "my-collection",
			wantErr: ErrInvalidName,
		},
		{
			name:    "name too long",
			colName: "this_is_a_very_long_collection_name_that_definitely_exceeds_the_maximum_allowed_length_of_64_characters_for_sure",
			wantErr: ErrNameTooLong,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateName(tt.colName)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCollection_Validate(t *testing.T) {
	tests := []struct {
		name       string
		collection *Collection
		wantErr    error
	}{
		{
			name: "valid collection",
			collection: &Collection{
				Namespace: "community",
				Name:      "general",
				Version:   "1.0.0",
			},
			wantErr: nil,
		},
		{
			name: "invalid namespace",
			collection: &Collection{
				Namespace: "Invalid-Namespace",
				Name:      "general",
				Version:   "1.0.0",
			},
			wantErr: ErrInvalidNamespace,
		},
		{
			name: "invalid name",
			collection: &Collection{
				Namespace: "community",
				Name:      "Invalid-Name",
				Version:   "1.0.0",
			},
			wantErr: ErrInvalidName,
		},
		{
			name: "invalid version",
			collection: &Collection{
				Namespace: "community",
				Name:      "general",
				Version:   "invalid",
			},
			wantErr: ErrInvalidVersion,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.collection.Validate()
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCollection_FullName(t *testing.T) {
	c := &Collection{
		Namespace: "community",
		Name:      "general",
	}

	assert.Equal(t, "community.general", c.FullName())
}

func TestCollection_Filename(t *testing.T) {
	c := &Collection{
		Namespace: "community",
		Name:      "general",
		Version:   "6.0.0",
	}

	assert.Equal(t, "community-general-6.0.0.tar.gz", c.Filename())
}
