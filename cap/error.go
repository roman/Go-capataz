package cap

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/capatazlib/go-capataz/internal/c"
)

// terminateNodeError is the error reported back to a Supervisor when the
// termination of a node fails
type terminateNodeError = error

// startNodeError is the error reported back to a Supervisor when the start of a
// node fails
type startNodeError = error

// ErrKVs is an utility interface used to get key-values out of Capataz errors
type ErrKVs interface {
	KVs() map[string]interface{}
}

// SupervisorTerminationError wraps errors returned by a child node that failed
// to terminate (io errors, timeouts, etc.), enhancing it with supervisor
// information
type SupervisorTerminationError struct {
	supRuntimeName string
	nodeErrMap     map[string]error
	rscCleanupErr  error
}

// Error returns an error message
func (err *SupervisorTerminationError) Error() string {
	return "supervisor terminated with failures"
}

// KVs returns a metadata map for structured logging
func (err *SupervisorTerminationError) KVs() map[string]interface{} {
	nodeNames := make([]string, 0, len(err.nodeErrMap))
	for nodeName := range err.nodeErrMap {
		nodeNames = append(nodeNames, nodeName)
	}
	sort.Strings(nodeNames)

	acc := make(map[string]interface{})
	acc["supervisor.name"] = err.supRuntimeName

	for i, nodeName := range nodeNames {
		nodeErr := err.nodeErrMap[nodeName]
		var subTreeError ErrKVs
		if errors.As(nodeErr, &subTreeError) {
			for k0, v := range subTreeError.KVs() {
				k := strings.TrimPrefix(k0, "supervisor.")
				acc[fmt.Sprintf("supervisor.subtree.%d.%s", i, k)] = v
			}
		} else {
			acc[fmt.Sprintf("supervisor.termination.node.%d.name", i)] = nodeName
			acc[fmt.Sprintf("supervisor.termination.node.%d.error", i)] = nodeErr
		}

	}

	if err.rscCleanupErr != nil {
		acc["supervisor.termination.cleanup.error"] = err.rscCleanupErr
	}

	return acc
}

// SupervisorBuildError wraps errors returned from a client provided function
// that builds the supervisor nodes, enhancing it with supervisor information
type SupervisorBuildError struct {
	supRuntimeName string
	buildNodesErr  error
}

func (err *SupervisorBuildError) Error() string {
	return "supervisor build nodes function failed"
}

// KVs returns a metadata map for structured logging
func (err *SupervisorBuildError) KVs() map[string]interface{} {
	acc := make(map[string]interface{})
	acc["supervisor.name"] = err.supRuntimeName
	acc["supervisor.build.error"] = err.buildNodesErr
	return acc
}

// SupervisorStartError wraps an error reported on the initialization of a child
// node, enhancing it with supervisor information and possible termination errors
// on other siblings
type SupervisorStartError struct {
	supRuntimeName string
	nodeName       string
	nodeErr        error
	terminationErr *SupervisorTerminationError
}

// Error returns an error message
func (err *SupervisorStartError) Error() string {
	return "supervisor node failed to start"
}

// KVs returns a metadata map for structured logging
func (err *SupervisorStartError) KVs() map[string]interface{} {
	acc := make(map[string]interface{})
	acc["supervisor.name"] = err.supRuntimeName

	if err.nodeErr != nil {
		var subTreeError ErrKVs
		if errors.As(err.nodeErr, &subTreeError) {
			for k0, v := range subTreeError.KVs() {
				k := strings.TrimPrefix(k0, "supervisor.")
				acc[fmt.Sprintf("supervisor.subtree.%s", k)] = v
			}
		} else {
			acc["supervisor.start.node.name"] = err.nodeName
			acc["supervisor.start.node.error"] = err.nodeErr
		}
	}

	if err.terminationErr != nil {
		for k, v := range err.terminationErr.KVs() {
			acc[k] = v
		}
	}

	return acc
}

// SupervisorRestartError wraps an error tolerance surpassed error from a child
// node, enhancing it with supervisor information and possible termination errors
// on other siblings
type SupervisorRestartError struct {
	supRuntimeName string
	nodeErr        *c.ErrorToleranceReached
	terminationErr *SupervisorTerminationError
}

// Error returns an error message
func (err *SupervisorRestartError) Error() string {
	return "supervisor crashed due to error tolerance surpassed"
}

// KVs returns a metadata map for structured logging
func (err *SupervisorRestartError) KVs() map[string]interface{} {
	acc := make(map[string]interface{})
	acc["supervisor.name"] = err.supRuntimeName

	if err.nodeErr != nil {
		for k, v := range err.nodeErr.KVs() {
			acc[fmt.Sprintf("supervisor.restart.%s", k)] = v
		}
	}

	if err.terminationErr != nil {
		for k, v := range err.terminationErr.KVs() {
			acc[k] = v
		}
	}

	return acc
}
