package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

func newCatalogCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "catalog",
		Short: "Catalog operations",
	}
	cmd.AddCommand(&cobra.Command{
		Use:   "list",
		Short: "List catalogs",
		RunE:  runCatalogList,
	})
	itemsCmd := &cobra.Command{
		Use:   "items",
		Short: "List items in a catalog",
		RunE:  runCatalogItems,
	}
	itemsCmd.Flags().StringP("catalog", "c", "", "Catalog name")
	cmd.AddCommand(itemsCmd)
	return cmd
}

func runCatalogList(cmd *cobra.Command, args []string) error {
	cl, _, err := getClientFromContext()
	if err != nil {
		return err
	}
	org, err := cl.GetOrg()
	if err != nil {
		return err
	}
	catalogs, err := org.QueryCatalogList()
	if err != nil {
		return err
	}
	for _, c := range catalogs {
		fmt.Printf("%-30s templates:%-3d media:%-3d %s\n", c.Name, c.NumberOfVAppTemplates, c.NumberOfMedia, c.Description)
	}
	return nil
}

func runCatalogItems(cmd *cobra.Command, args []string) error {
	catalogName, _ := cmd.Flags().GetString("catalog")
	if catalogName == "" {
		if len(args) < 1 {
			return fmt.Errorf("catalog name required (use --catalog or pass as argument)")
		}
		catalogName = args[0]
	}

	cl, _, err := getClientFromContext()
	if err != nil {
		return err
	}
	org, err := cl.GetOrg()
	if err != nil {
		return err
	}
	catalog, err := org.GetCatalogByName(catalogName, false)
	if err != nil {
		return fmt.Errorf("finding catalog %q: %w", catalogName, err)
	}
	for _, ci := range catalog.Catalog.CatalogItems {
		for _, item := range ci.CatalogItem {
			fullItem, err := catalog.GetCatalogItemByHref(item.HREF)
			entityType := "unknown"
			status := -1
			if err == nil {
				if tpl, err := fullItem.GetVAppTemplate(); err == nil {
					entityType = "template"
					status = tpl.VAppTemplate.Status
				} else {
					entityType = "media"
				}
			}
			fmt.Printf("%-45s %-10s status:%d\n", item.Name, entityType, status)
		}
	}
	return nil
}

func init() {
	rootCmd.AddCommand(newCatalogCmd())
}
