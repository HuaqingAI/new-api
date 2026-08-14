package agentplatform

import (
	"archive/zip"
	"bytes"
	"errors"
	"testing"

	apmodel "github.com/QuantumNous/new-api/model/agentplatform"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newSkillServiceForTest(t *testing.T) (*SkillService, *gorm.DB) {
	t.Helper()

	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, apmodel.Migrate(db))
	return NewSkillService(db), db
}

func TestSkillServiceCreatesListsAndUpdatesOnlySkills(t *testing.T) {
	svc, db := newSkillServiceForTest(t)

	created, err := svc.Create(SkillCreateInput{
		DisplayName: "My Skill",
		OwnerUserId: 101,
		TenantId:    7,
	})
	require.NoError(t, err)
	require.Equal(t, apmodel.ResourceTypeSkill, created.ResourceType)

	require.NoError(t, db.Create(&apmodel.Resource{
		ResourceType: apmodel.ResourceTypeKnowledge,
		DisplayName:  "Other Resource",
		OwnerUserId:  101,
	}).Error)

	list, err := svc.List(SkillQuery{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, 1, list.Total)
	require.Len(t, list.Items, 1)
	require.Equal(t, created.ResourceId, list.Items[0].ResourceId)

	updated, err := svc.Update(created.ResourceId, SkillUpdateInput{DisplayName: "My Skill V2"})
	require.NoError(t, err)
	require.Equal(t, "My Skill V2", updated.DisplayName)
}

func TestSkillServiceGetRejectsNonSkillResources(t *testing.T) {
	svc, db := newSkillServiceForTest(t)

	resource := apmodel.Resource{
		ResourceType: apmodel.ResourceTypeAgent,
		DisplayName:  "Agent",
		OwnerUserId:  100,
	}
	require.NoError(t, db.Create(&resource).Error)

	_, err := svc.Get(resource.ResourceId)
	require.ErrorIs(t, err, ErrResourceNotFound)
}

func TestSkillServiceSavePackageAcceptsSingleRootDirectoryIndependentOfDisplayName(t *testing.T) {
	svc, _ := newSkillServiceForTest(t)
	store := newFakeArtifactStore()
	restore := SetArtifactStoreForTest(store)
	t.Cleanup(restore)
	skill, err := svc.Create(SkillCreateInput{DisplayName: "1111", OwnerUserId: 100})
	require.NoError(t, err)

	skillZip := buildTestSkillZip(t, "refund-order-reconciler/SKILL.md", "# refund")
	saved, err := svc.SavePackage(skill.ResourceId, "refund-order-reconciler.zip", bytes.NewReader(skillZip))
	require.NoError(t, err)

	require.Equal(t, "refund-order-reconciler.zip", saved.FileName)
	require.NotEmpty(t, saved.FilePath)
	require.Len(t, store.files, 1)
}

func TestSkillServiceSavePackageRejectsZipWithoutSingleRootDirectory(t *testing.T) {
	for name, skillZip := range map[string][]byte{
		"flat.zip":       buildTestSkillZip(t, "SKILL.md", "# invalid"),
		"multi-root.zip": buildTestSkillZipWithFiles(t, map[string]string{"a/SKILL.md": "# a", "b/SKILL.md": "# b"}),
	} {
		t.Run(name, func(t *testing.T) {
			svc, _ := newSkillServiceForTest(t)
			restore := SetArtifactStoreForTest(newFakeArtifactStore())
			t.Cleanup(restore)
			skill, err := svc.Create(SkillCreateInput{DisplayName: "1111", OwnerUserId: 100})
			require.NoError(t, err)

			_, err = svc.SavePackage(skill.ResourceId, name, bytes.NewReader(skillZip))

			require.ErrorIs(t, err, ErrInvalidResourceInput)
			var def apmodel.SkillDef
			require.True(t, errors.Is(svc.db.Where("resource_id = ?", skill.ResourceId).First(&def).Error, gorm.ErrRecordNotFound))
		})
	}
}

func buildTestSkillZipWithFiles(t *testing.T, files map[string]string) []byte {
	t.Helper()
	buffer := bytes.NewBuffer(nil)
	writer := zip.NewWriter(buffer)
	for name, content := range files {
		file, err := writer.Create(name)
		require.NoError(t, err)
		_, err = file.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	return buffer.Bytes()
}
