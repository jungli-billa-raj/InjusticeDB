package db

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/jungli-billa-raj/InjusticeDB/internal/models"
)

// setupCulpritTestDB prepares a pool and truncates related tables for isolation.
func setupCulpritTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgrespassword@localhost:5432/injusticedb?sslmode=disable"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err, "failed to connect to test database")

	_, err = pool.Exec(ctx, "TRUNCATE TABLE people, incidents, incident_culprits CASCADE;")
	require.NoError(t, err, "failed to truncate tables")

	teardown := func() {
		pool.Close()
	}

	return pool, teardown
}

func TestPostgresCulpritRepository_CreatePerson(t *testing.T) {
	pool, teardown := setupCulpritTestDB(t)
	defer teardown()

	repo := NewPostgresCulpritRepository(pool)
	ctx := context.Background()

	t.Run("Create person with full details successfully", func(t *testing.T) {
		org := "Local Mining Syndicate"
		age := 42
		state := "Jharkhand"
		city := "Dhanbad"

		personInput := models.Person{
			Name:         "John Doe",
			Organization: &org,
			Age:          &age,
			State:        &state,
			City:         &city,
		}

		created, err := repo.CreatePerson(ctx, personInput)
		require.NoError(t, err)
		require.NotNil(t, created)

		assert.NotEqual(t, uuid.Nil, created.ID)
		assert.Equal(t, "John Doe", created.Name)
		assert.Equal(t, &org, created.Organization)
		assert.Equal(t, &age, created.Age)
		assert.Equal(t, &state, created.State)
		assert.Equal(t, &city, created.City)
		assert.False(t, created.CreatedAt.IsZero())
	})

	t.Run("Create person with minimal details (nil optional fields)", func(t *testing.T) {
		personInput := models.Person{
			Name: "Unknown Suspect",
		}

		created, err := repo.CreatePerson(ctx, personInput)
		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, created.ID)
		assert.Equal(t, "Unknown Suspect", created.Name)
		assert.Nil(t, created.Organization)
		assert.Nil(t, created.Age)
	})
}

func TestPostgresCulpritRepository_LinkAndGetForIncident(t *testing.T) {
	pool, teardown := setupCulpritTestDB(t)
	defer teardown()

	incidentRepo := NewPostgresIncidentRepository(pool)
	culpritRepo := NewPostgresCulpritRepository(pool)
	ctx := context.Background()

	// Seed Parent Incident
	inc, err := incidentRepo.Create(ctx, models.CreateIncidentParams{
		Title:         "Culprit Link Incident",
		FullStory:     "Testing linking suspects to incidents.",
		Severity:      8,
		JusticeStatus: models.JusticeProceeding,
		State:         "Jharkhand",
		City:          "Ranchi",
	})
	require.NoError(t, err)

	// Seed Person
	org := "Corrupt Firm"
	person, err := culpritRepo.CreatePerson(ctx, models.Person{
		Name:         "Jane Smith",
		Organization: &org,
	})
	require.NoError(t, err)

	t.Run("LinkToIncident adds culprit to junction table and GetCulpritsForIncident fetches it", func(t *testing.T) {
		err := culpritRepo.LinkToIncident(ctx, inc.ID, person.ID, models.CulpritSuspect)
		require.NoError(t, err)

		culprits, err := culpritRepo.GetCulpritsForIncident(ctx, inc.ID)
		require.NoError(t, err)
		require.Len(t, culprits, 1)

		assert.Equal(t, inc.ID, culprits[0].IncidentID)
		assert.Equal(t, person.ID, culprits[0].PersonID)
		assert.Equal(t, models.CulpritSuspect, culprits[0].CulpritStatus)

		// Verify joined Person metadata
		assert.Equal(t, person.ID, culprits[0].Person.ID)
		assert.Equal(t, "Jane Smith", culprits[0].Person.Name)
		assert.Equal(t, &org, culprits[0].Person.Organization)
	})

	t.Run("LinkToIncident upserts status on conflict", func(t *testing.T) {
		// Relink same person with updated status 'accused'
		err := culpritRepo.LinkToIncident(ctx, inc.ID, person.ID, models.CulpritAccused)
		require.NoError(t, err)

		culprits, err := culpritRepo.GetCulpritsForIncident(ctx, inc.ID)
		require.NoError(t, err)
		require.Len(t, culprits, 1)
		assert.Equal(t, models.CulpritAccused, culprits[0].CulpritStatus)
	})
}

func TestPostgresCulpritRepository_UpdateCulpritStatus(t *testing.T) {
	pool, teardown := setupCulpritTestDB(t)
	defer teardown()

	incidentRepo := NewPostgresIncidentRepository(pool)
	culpritRepo := NewPostgresCulpritRepository(pool)
	ctx := context.Background()

	inc, err := incidentRepo.Create(ctx, models.CreateIncidentParams{
		Title:         "Status Update Incident",
		FullStory:     "Story for legal tracking.",
		Severity:      9,
		JusticeStatus: models.JusticeProceeding,
		State:         "Bihar",
		City:          "Patna",
	})
	require.NoError(t, err)

	person, err := culpritRepo.CreatePerson(ctx, models.Person{
		Name: "Robert Paulson",
	})
	require.NoError(t, err)

	err = culpritRepo.LinkToIncident(ctx, inc.ID, person.ID, models.CulpritSuspect)
	require.NoError(t, err)

	t.Run("Update status to convicted successfully", func(t *testing.T) {
		err := culpritRepo.UpdateCulpritStatus(ctx, inc.ID, person.ID, models.CulpritConvicted)
		require.NoError(t, err)

		culprits, err := culpritRepo.GetCulpritsForIncident(ctx, inc.ID)
		require.NoError(t, err)
		require.Len(t, culprits, 1)
		assert.Equal(t, models.CulpritConvicted, culprits[0].CulpritStatus)
	})

	t.Run("Update status returns error for unlinked culprit-incident pair", func(t *testing.T) {
		unlinkedPersonID := uuid.New()
		err := culpritRepo.UpdateCulpritStatus(ctx, inc.ID, unlinkedPersonID, models.CulpritGuilty)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "culprit-incident link not found")
	})
}
