package cmds

import (
	"testing"

	"github.com/pachyderm/pachyderm/v2/src/internal/require"
	tu "github.com/pachyderm/pachyderm/v2/src/internal/testutil"
)

func TestConnectorCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	tu.ActivateAuth(t)
	defer tu.DeleteAll(t)
	require.NoError(t, tu.BashCmd(`
		echo '{"id": "{{.id}}", "name": "testconn", "type": "github", "config": {"id": 1234}}' | pachctl idp create-connector 
		pachctl idp list-connector | match '{{.id}}'
		pachctl idp get-connector {{.id}} \
		  | match 'Name: testconn' \
		  | match 'Type: github' \
		  | match 'Version: 0' \
                  | match '    id: 1234' 
		echo '{"id": "{{.id}}", "version": 1, "config": {"client_id": "a"}}' | pachctl idp update-connector 
		pachctl idp get-connector {{.id}} \
		  | match 'Name: testconn' \
		  | match 'Type: github' \
		  | match 'Version: 1' \
		  | match '    client_id: a'
		echo '{"id": "{{.id}}", "version": 2, "name": "newname2"}' | pachctl idp update-connector
		pachctl idp get-connector {{.id}} \
		  | match 'Name: newname2' \
		  | match 'Type: github' \
		  | match 'Version: 2' \
		  | match '    client_id: a'
		pachctl idp delete-connector {{.id}}
		`,
		"id", tu.UniqueString("connector"),
	).Run())
}

func TestClientCRUD(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	tu.ActivateAuth(t)
	defer tu.DeleteAll(t)
	require.NoError(t, tu.BashCmd(`
		pachctl idp create-client --id {{.id}} --name 'testclient' --secret 'a secret' --redirectUris https://localhost:1234 \
		  | match 'secret: "a secret"'
		pachctl idp list-client | match '{{.id}}'
		pachctl idp get-client {{.id}} \
		  | match 'name: testclient' \
		  | match 'secret: a secret' \
		  | match 'redirect URIs: https://localhost:1234' \
		  | match 'trusted peers: ' 
		pachctl idp update-client {{.id}} --name 'newname' --redirectUris https://localhost:1234,https://localhost:5678 --trustedPeers x,y,z
		pachctl idp get-client {{.id}} \
		  | match 'name: newname' \
		  | match 'secret: a secret' \
		  | match 'redirect URIs: https://localhost:1234, https://localhost:5678' \
		  | match 'trusted peers: x, y, z' 
		pachctl idp update-client {{.id}} --name 'newname2'
		pachctl idp get-client {{.id}} \
	  	  | match 'name: newname2' \
		  | match 'secret: a secret' \
		  | match 'redirect URIs: https://localhost:1234, https://localhost:5678' \
		  | match 'trusted peers: x, y, z' 
		pachctl idp delete-client {{.id}}
		`,
		"id", tu.UniqueString("client"),
	).Run())
}

func TestGetSetConfig(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}
	tu.ActivateAuth(t)
	defer tu.DeleteAll(t)
	require.NoError(t, tu.BashCmd(`
		pachctl idp set-config --issuer 'http://example.com:1234'
		pachctl idp get-config | match 'issuer: "http://example.com:1234"' 
		`,
		"id", tu.UniqueString("connector"),
	).Run())
}
