//go:build windows

package motionplan

import (
	"github.com/edaniels/golog"
	"github.com/pkg/errors"

	"go.viam.com/rdk/referenceframe"
)

// CreateCombinedIKSolver is not supported on windows.
// TODO(RSDK-1772): support motion planning on windows
func CreateCombinedIKSolver(model referenceframe.Frame, logger golog.Logger, nCPU int) (InverseKinematics, error) {
	return nil, errors.New("motion planning is not yet supported on Windows")
}
