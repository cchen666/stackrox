package postgres

//go:generate pg-table-bindings-wrapper --type=storage.IntegrationHealth --table=integrationhealth --permission-checker permissionCheckerSingleton()
