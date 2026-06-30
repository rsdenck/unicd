package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newVmMediaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "media",
		Short: "VM media (ISO) operations",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "insert <vm> <catalog> <media>",
		Short: "Insert/attach an ISO to a VM",
		Args:  cobra.ExactArgs(3),
		RunE:  runVmMediaInsert,
	})
	cmd.AddCommand(&cobra.Command{
		Use:   "eject <vm> <catalog> <media>",
		Short: "Eject/detach ISO from a VM",
		Args:  cobra.ExactArgs(3),
		RunE:  runVmMediaEject,
	})
	return cmd
}

func runVmMediaInsert(cmd *cobra.Command, args []string) error {
	vmName, catalogName, mediaName := args[0], args[1], args[2]
	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	govm, err := findVM(cl, ctx, vmName)
	if err != nil {
		return err
	}
	org, err := cl.GetOrg()
	if err != nil {
		return err
	}
	task, err := govm.HandleInsertMedia(org, catalogName, mediaName)
	if err != nil {
		return fmt.Errorf("inserting media: %w", err)
	}
	if err := task.WaitTaskCompletion(); err != nil {
		return err
	}
	fmt.Printf("ISO %q from catalog %q inserted into VM %q\n", mediaName, catalogName, vmName)
	return nil
}

func runVmMediaEject(cmd *cobra.Command, args []string) error {
	vmName, catalogName, mediaName := args[0], args[1], args[2]
	cl, ctx, err := getClientFromContext()
	if err != nil {
		return err
	}
	govm, err := findVM(cl, ctx, vmName)
	if err != nil {
		return err
	}
	org, err := cl.GetOrg()
	if err != nil {
		return err
	}
	ejectTask, err := govm.HandleEjectMedia(org, catalogName, mediaName)
	if err != nil {
		return fmt.Errorf("ejecting media: %w", err)
	}
	if err := ejectTask.WaitTaskCompletion(false); err != nil {
		return err
	}
	fmt.Printf("ISO ejected from VM %q\n", vmName)
	return nil
}
