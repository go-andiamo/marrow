package marrow

import (
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestAuth(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		a := Auth(BasicAuth, Var("foo"))
		ctx := newTestContext(map[Var]any{"foo": "bar"})
		v, err := a.ResolveValue(ctx)
		require.NoError(t, err)
		assert.Equal(t, "Basic bar", v)
		assert.Equal(t, "Auth(\"Basic\", Var(foo))", fmt.Sprintf("%s", a))
	})
	t.Run("bearer", func(t *testing.T) {
		a := Auth(BearerAuth, Var("foo"))
		ctx := newTestContext(map[Var]any{"foo": "bar"})
		v, err := a.ResolveValue(ctx)
		require.NoError(t, err)
		assert.Equal(t, "Bearer bar", v)
		assert.Equal(t, "Auth(\"Bearer\", Var(foo))", fmt.Sprintf("%s", a))
	})
	t.Run("token", func(t *testing.T) {
		a := Auth(TokenAuth, Var("foo"))
		ctx := newTestContext(map[Var]any{"foo": "bar"})
		v, err := a.ResolveValue(ctx)
		require.NoError(t, err)
		assert.Equal(t, "Token bar", v)
		assert.Equal(t, "Auth(\"Token\", Var(foo))", fmt.Sprintf("%s", a))
	})
	t.Run("other", func(t *testing.T) {
		a := Auth(AuthScheme("Other"), Var("foo"))
		ctx := newTestContext(map[Var]any{"foo": 42})
		v, err := a.ResolveValue(ctx)
		require.NoError(t, err)
		assert.Equal(t, "Other 42", v)
		assert.Equal(t, "Auth(\"Other\", Var(foo))", fmt.Sprintf("%s", a))
	})
	t.Run("empty", func(t *testing.T) {
		a := Auth("", Var("foo"))
		ctx := newTestContext(map[Var]any{"foo": "bar"})
		v, err := a.ResolveValue(ctx)
		require.NoError(t, err)
		assert.Equal(t, "bar", v)
		assert.Equal(t, "Auth(\"\", Var(foo))", fmt.Sprintf("%s", a))
	})
	t.Run("errors", func(t *testing.T) {
		a := Auth("", Var("foo"))
		ctx := newTestContext(nil)
		_, err := a.ResolveValue(ctx)
		require.Error(t, err)
	})
}

func TestAuthUsernamePassword(t *testing.T) {
	t.Run("basic", func(t *testing.T) {
		a := AuthUsernamePassword(Var("username"), Var("password"))
		ctx := newTestContext(map[Var]any{"username": "foo", "password": "bar"})
		v, err := a.ResolveValue(ctx)
		require.NoError(t, err)
		assert.Equal(t, "Zm9vOmJhcg==", v)
		assert.Equal(t, "AuthUsernamePassword(Var(username), Var(password))", fmt.Sprintf("%s", a))
	})
	t.Run("errors 1", func(t *testing.T) {
		a := AuthUsernamePassword(Var("username"), Var("password"))
		ctx := newTestContext(map[Var]any{"username": "foo"})
		_, err := a.ResolveValue(ctx)
		require.Error(t, err)
	})
	t.Run("errors 2", func(t *testing.T) {
		a := AuthUsernamePassword(Var("username"), Var("password"))
		ctx := newTestContext(nil)
		_, err := a.ResolveValue(ctx)
		require.Error(t, err)
	})
}

func TestJwt(t *testing.T) {
	a := Jwt(Var("secret"),
		SubjectClaim(Var("user")),
		IssuerClaim(Var("issuer")),
		AudienceClaim(Var("audience")),
		ExpireAfterClaim(10*time.Minute),
		Claim("foo", "bar"))
	ctx := newTestContext(map[Var]any{
		"secret":   "my-secret",
		"user":     "my-user",
		"issuer":   "my-issuer",
		"audience": "my-audience",
	})
	v, err := a.ResolveValue(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, v)
	tokenStr := v.(string)
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return []byte("my-secret"), nil
	})
	require.NoError(t, err)
	assert.NotNil(t, token)
	claims := token.Claims.(jwt.MapClaims)
	assert.Equal(t, "my-user", claims["sub"])
	assert.Equal(t, "my-issuer", claims["iss"])
	assert.Equal(t, "my-audience", claims["aud"])
	assert.Equal(t, "bar", claims["foo"])
}

func TestJwtHS384(t *testing.T) {
	a := JwtHS384(Var("secret"),
		SubjectClaim(Var("user")),
		IssuerClaim(Var("issuer")),
		AudienceClaim(Var("audience")),
		ExpireAtClaim(time.Now().Add(time.Hour)),
		Claim("foo", "bar"))
	ctx := newTestContext(map[Var]any{
		"secret":   []byte("my-secret"),
		"user":     "my-user",
		"issuer":   "my-issuer",
		"audience": "my-audience",
	})
	v, err := a.ResolveValue(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, v)
	tokenStr := v.(string)
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return []byte("my-secret"), nil
	})
	require.NoError(t, err)
	assert.NotNil(t, token)
	claims := token.Claims.(jwt.MapClaims)
	assert.Equal(t, "my-user", claims["sub"])
	assert.Equal(t, "my-issuer", claims["iss"])
	assert.Equal(t, "my-audience", claims["aud"])
	assert.Equal(t, "bar", claims["foo"])
}

func TestJwtHS512(t *testing.T) {
	a := JwtHS512(Var("secret"),
		SubjectClaim(Var("user")),
		IssuerClaim(Var("issuer")),
		AudienceClaim(Var("audience")),
		ExpireAtClaim(time.Now().Add(time.Hour)),
		Claim("foo", "bar"))
	ctx := newTestContext(map[Var]any{
		"secret":   42,
		"user":     "my-user",
		"issuer":   "my-issuer",
		"audience": "my-audience",
	})
	v, err := a.ResolveValue(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, v)
	tokenStr := v.(string)
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return []byte("42"), nil
	})
	require.NoError(t, err)
	assert.NotNil(t, token)
	claims := token.Claims.(jwt.MapClaims)
	assert.Equal(t, "my-user", claims["sub"])
	assert.Equal(t, "my-issuer", claims["iss"])
	assert.Equal(t, "my-audience", claims["aud"])
	assert.Equal(t, "bar", claims["foo"])
}
