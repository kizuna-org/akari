package usecase

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/kizuna-org/akari/kiseki/pkg/character/domain/entity"
)

// mockCharacterRepository is a mock implementation for testing
type mockCharacterRepository struct {
	createFunc func(ctx context.Context, character *entity.Character) error
	getFunc    func(ctx context.Context, id uuid.UUID) (*entity.Character, error)
	listFunc   func(ctx context.Context) ([]*entity.Character, error)
	updateFunc func(ctx context.Context, character *entity.Character) error
	deleteFunc func(ctx context.Context, id uuid.UUID) error
	existsFunc func(ctx context.Context, id uuid.UUID) (bool, error)
}

func (m *mockCharacterRepository) Create(ctx context.Context, character *entity.Character) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, character)
	}
	return nil
}

func (m *mockCharacterRepository) Get(ctx context.Context, id uuid.UUID) (*entity.Character, error) {
	if m.getFunc != nil {
		return m.getFunc(ctx, id)
	}
	return entity.NewCharacter("test"), nil
}

func (m *mockCharacterRepository) List(ctx context.Context) ([]*entity.Character, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx)
	}
	return []*entity.Character{}, nil
}

func (m *mockCharacterRepository) Update(ctx context.Context, character *entity.Character) error {
	if m.updateFunc != nil {
		return m.updateFunc(ctx, character)
	}
	return nil
}

func (m *mockCharacterRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id)
	}
	return nil
}

func (m *mockCharacterRepository) Exists(ctx context.Context, id uuid.UUID) (bool, error) {
	if m.existsFunc != nil {
		return m.existsFunc(ctx, id)
	}
	return true, nil
}

func TestCharacterInteractor_CreateCharacter(t *testing.T) {
	tests := []struct {
		name    string
		input   CreateCharacterInput
		wantErr bool
	}{
		{
			name: "successful creation",
			input: CreateCharacterInput{
				Name: "Test Character",
			},
			wantErr: false,
		},
		{
			name: "empty name",
			input: CreateCharacterInput{
				Name: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockCharacterRepository{}
			interactor := NewCharacterInteractor(repo)

			output, err := interactor.CreateCharacter(context.Background(), tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreateCharacter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if output == nil {
					t.Fatal("CreateCharacter() returned nil output")
				}
				if output.Character.Name != tt.input.Name {
					t.Errorf("Character.Name = %v, want %v", output.Character.Name, tt.input.Name)
				}
			}
		})
	}
}

func TestCharacterInteractor_GetCharacter(t *testing.T) {
	characterID := uuid.New()

	tests := []struct {
		name    string
		input   GetCharacterInput
		wantErr bool
	}{
		{
			name: "successful get",
			input: GetCharacterInput{
				ID: characterID,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockCharacterRepository{}
			interactor := NewCharacterInteractor(repo)

			output, err := interactor.GetCharacter(context.Background(), tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetCharacter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && output == nil {
				t.Fatal("GetCharacter() returned nil output")
			}
		})
	}
}

func TestCharacterInteractor_ListCharacters(t *testing.T) {
	tests := []struct {
		name      string
		mockChars []*entity.Character
		wantCount int
	}{
		{
			name:      "empty list",
			mockChars: []*entity.Character{},
			wantCount: 0,
		},
		{
			name: "list with characters",
			mockChars: []*entity.Character{
				entity.NewCharacter("Char 1"),
				entity.NewCharacter("Char 2"),
			},
			wantCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockCharacterRepository{
				listFunc: func(ctx context.Context) ([]*entity.Character, error) {
					return tt.mockChars, nil
				},
			}
			interactor := NewCharacterInteractor(repo)

			output, err := interactor.ListCharacters(context.Background())
			if err != nil {
				t.Errorf("ListCharacters() error = %v", err)
				return
			}

			if len(output.Characters) != tt.wantCount {
				t.Errorf("ListCharacters() returned %d characters, want %d", len(output.Characters), tt.wantCount)
			}
		})
	}
}

func TestCharacterInteractor_UpdateCharacter(t *testing.T) {
	characterID := uuid.New()

	tests := []struct {
		name    string
		input   UpdateCharacterInput
		wantErr bool
	}{
		{
			name: "successful update",
			input: UpdateCharacterInput{
				ID:   characterID,
				Name: "Updated Name",
			},
			wantErr: false,
		},
		{
			name: "empty name",
			input: UpdateCharacterInput{
				ID:   characterID,
				Name: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockCharacterRepository{}
			interactor := NewCharacterInteractor(repo)

			output, err := interactor.UpdateCharacter(context.Background(), tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateCharacter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if output == nil {
					t.Fatal("UpdateCharacter() returned nil output")
				}
				if output.Character.Name != tt.input.Name {
					t.Errorf("Character.Name = %v, want %v", output.Character.Name, tt.input.Name)
				}
			}
		})
	}
}

func TestCharacterInteractor_DeleteCharacter(t *testing.T) {
	characterID := uuid.New()

	tests := []struct {
		name    string
		input   DeleteCharacterInput
		wantErr bool
	}{
		{
			name: "successful delete",
			input: DeleteCharacterInput{
				ID: characterID,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &mockCharacterRepository{}
			interactor := NewCharacterInteractor(repo)

			err := interactor.DeleteCharacter(context.Background(), tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteCharacter() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
