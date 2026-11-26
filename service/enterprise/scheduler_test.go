package enterprise

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEnterpriseAlertDispatchTickerSupportsThirtySecondRetryWindow(t *testing.T) {
	require.Equal(t, 30*time.Second, enterpriseAlertDispatchTaskTickInterval)
	require.LessOrEqual(t, enterpriseAlertDispatchTaskTickInterval, 30*time.Second)
	require.Greater(t, enterpriseMaintenanceTaskTickInterval, enterpriseAlertDispatchTaskTickInterval)
}
