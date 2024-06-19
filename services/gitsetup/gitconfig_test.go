package gitsetup

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockGoDotEnv mocks the GoDotEnvLoader interface.
type MockGoDotEnv struct {
	mock.Mock
}

// Load mocks the Load method.
func (m *MockGoDotEnv) Load(filenames ...string) error {
	args := m.Called(filenames[0])
	return args.Error(0)
}

// MockOS mocks the OSGetter interface.
type MockOS struct {
	mock.Mock
}

// Getenv mocks the Getenv method.
func (m *MockOS) Getenv(key string) string {
	args := m.Called(key)
	return args.String(0)
}

var (
	mockGoDotEnv *MockGoDotEnv
	mockOS       *MockOS
)

func setupMocks() {
	mockGoDotEnv = new(MockGoDotEnv)
	mockOS = new(MockOS)
	goDotEnvLoader = mockGoDotEnv
	osGetter = mockOS
}

func TestLoadEnv_Success(t *testing.T) {
	setupMocks()

	// Arrange
	mockGoDotEnv.On("Load", ".env").Return(nil)
	mockOS.On("Getenv", "TEMPLATE_URL").Return("some_template_url")

	// Act & Assert
	assert.NotPanics(t, func() {
		loadEnv()
	})

	// Assert
	mockGoDotEnv.AssertCalled(t, "Load", ".env")
	mockOS.AssertCalled(t, "Getenv", "TEMPLATE_URL")
}

func TestLoadEnv_Failure(t *testing.T) {
	setupMocks()

	// Arrange
	mockGoDotEnv.On("Load", ".env").Return(nil)
	mockOS.On("Getenv", "TEMPLATE_URL").Return("")

	// Act & Assert
	assert.PanicsWithValue(t, "TEMPLATE_URL must be set in the environment", func() {
		loadEnv()
	})

	// Assert
	mockGoDotEnv.AssertCalled(t, "Load", ".env")
	mockOS.AssertCalled(t, "Getenv", "TEMPLATE_URL")
}

func TestCheckTemplateURL_Success(t *testing.T) {
	setupMocks()

	// Arrange
	mockOS.On("Getenv", "TEMPLATE_URL").Return("some_template_url")

	// Act & Assert
	assert.NotPanics(t, func() {
		checkTemplateURL()
	})

	// Assert
	mockOS.AssertCalled(t, "Getenv", "TEMPLATE_URL")
}

func TestCheckTemplateURL_Failure(t *testing.T) {
	setupMocks()

	// Arrange
	mockOS.On("Getenv", "TEMPLATE_URL").Return("")

	// Act & Assert
	assert.PanicsWithValue(t, "TEMPLATE_URL must be set in the environment", func() {
		checkTemplateURL()
	})

	// Assert
	mockOS.AssertCalled(t, "Getenv", "TEMPLATE_URL")
}

func TestDefaultRepoConfig(t *testing.T) {
	setupMocks()

	// Arrange
	mockOS.On("Getenv", "TEMPLATE_URL").Return("some_template_url")

	// Act
	config := DefaultRepoConfig("repoName", "description")

	// Assert
	assert.Equal(t, "repoName", config.Name)
	assert.Equal(t, "description", config.Description)
	assert.Equal(t, true, config.Private)
	assert.Equal(t, true, config.AutoInit)
	assert.Equal(t, "some_template_url", config.TemplateURL)

	// Verify that Getenv was called with the correct parameter
	mockOS.AssertCalled(t, "Getenv", "TEMPLATE_URL")
}
