package ec2

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/spf13/cobra"

	"github.com/bharath-srinivas/nephele/cmd/nephele/command"
	"github.com/bharath-srinivas/nephele/function"
	"github.com/bharath-srinivas/nephele/internal/spinner"
)

// stop instance command.
var stopCmd = &cobra.Command{
	Use:     "stop [instance id]",
	Short:   "Stop the specified EC2 instance",
	Args:    cobra.MinimumNArgs(1),
	Example: "  nephele stop i-0a12b345c678de",
	PreRun:  command.PreRun,
	RunE:    stopInstance,
}

func init() {
	ec2Cmd.AddCommand(stopCmd)
	stopCmd.Flags().BoolVarP(&dryRun, "dry-run", "", false, "perform the operation with dry run enabled")
}

// run command.
func stopInstance(cmd *cobra.Command, args []string) error {
	sp := spinner.Default(spinner.Prefix[2])
	sp.Start()
	sess := ec2.New(command.Session)

	instanceId := function.EC2{
		IDs: args,
	}

	ec2Service := &function.EC2Service{
		EC2:     instanceId,
		Service: sess,
	}

	resp, err := ec2Service.StopInstances(dryRun)
	if err != nil {
		sp.Stop()
		return err
	}

	sp.Stop()
	for _, data := range resp.StoppingInstances {
		fmt.Println("Previous State(" + *data.InstanceId + ") : " + *data.PreviousState.Name)
		fmt.Println("Current State(" + *data.InstanceId + ")  : " + *data.CurrentState.Name)
		fmt.Printf("\n")
	}

	return nil
}
