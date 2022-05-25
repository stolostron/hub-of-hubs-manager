package policies

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestPolicyStatusList(t *testing.T) {
	// use the js encodeURIComponent() to encode password
	postgreUrl := "postgresql://hoh-process-user:password@host:5432/hoh"
	dbConnectionPool, err := pgxpool.Connect(context.Background(), postgreUrl)
	require.NoError(t, err)
	defer dbConnectionPool.Close()

	sql := sqlQuery()
	policiesStatus := []policyStatus{}
	handleRow(sql, dbConnectionPool, &policiesStatus)

	str, err := json.MarshalIndent(policiesStatus, "", "  ")
	require.NoError(t, err)
	t.Log(string(str))
}

func handleRow(query string, dbConnectionPool *pgxpool.Pool, policiesStatus *[]policyStatus) {
	rows, err := dbConnectionPool.Query(context.TODO(), query)
	if err != nil {
		panic(err)
	}

	for rows.Next() {
		var id, cluster_name, leaf_hub_name, errInfo, compliance string
		err := rows.Scan(&id, &cluster_name, &leaf_hub_name, &errInfo, &compliance)
		if err != nil {
			fmt.Fprintf(gin.DefaultWriter, "error in scanning a policy: %v\n", err)
			continue
		}
		policyStatus := policyStatus{
			PolicyId:    id,
			ClusterName: cluster_name,
			LeafHubName: leaf_hub_name,
			ErrorInfo:   errInfo,
			Compliance:  compliance,
		}
		*policiesStatus = append(*policiesStatus, policyStatus)
	}
}
