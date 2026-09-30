package enterprise

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	entmodel "github.com/QuantumNous/new-api/model/enterprise"
	"golang.org/x/sync/singleflight"
)

const (
	dingTalkScopeCacheTTL = time.Minute
	dingTalkScopeMaxDepth = 32
	dingTalkScopeMaxNodes = 10000
)

type DingTalkDirectoryClient interface {
	GetAccessToken(ctx context.Context, appKey string, appSecret string) (string, error)
	GetContactUserByUnionId(ctx context.Context, appToken string, unionId string) (DingTalkContactUserInfo, error)
	GetUserById(ctx context.Context, appToken string, userId string) (DingTalkDepartmentUserInfo, error)
	ListSubDepartments(ctx context.Context, appToken string, departmentId int64) ([]DingTalkDepartmentInfo, error)
}

type dingTalkScopeTree struct {
	Departments map[int64]DingTalkDepartmentInfo
	Order       []int64
}

type dingTalkScopeCacheEntry struct {
	tree      dingTalkScopeTree
	expiresAt time.Time
}

type DingTalkScopeResolver struct {
	mu    sync.Mutex
	cache map[string]dingTalkScopeCacheEntry
	group singleflight.Group
}

var defaultDingTalkScopeResolver = &DingTalkScopeResolver{
	cache: map[string]dingTalkScopeCacheEntry{},
}

func (r *DingTalkScopeResolver) Resolve(
	ctx context.Context,
	client DingTalkDirectoryClient,
	config entmodel.DingTalkConfig,
	appToken string,
) (dingTalkScopeTree, bool, error) {
	if client == nil {
		return dingTalkScopeTree{}, false, ErrDingTalkOAuthProviderFailed
	}
	key := dingTalkScopeCacheKey(config)
	if tree, ok := r.get(key); ok {
		return tree, true, nil
	}

	value, err, _ := r.group.Do(key, func() (any, error) {
		if tree, ok := r.get(key); ok {
			return tree, nil
		}
		tree, err := loadDingTalkScopeTree(ctx, client, appToken, config.SyncScope)
		if err != nil {
			return nil, err
		}
		r.put(key, tree)
		return tree, nil
	})
	if err != nil {
		return dingTalkScopeTree{}, false, err
	}
	tree, ok := value.(dingTalkScopeTree)
	if !ok {
		return dingTalkScopeTree{}, false, ErrDingTalkOAuthProviderFailed
	}
	return tree, false, nil
}

func (r *DingTalkScopeResolver) ClearTenant(tenantId int) {
	prefix := fmt.Sprintf("%d:", tenantId)
	r.mu.Lock()
	defer r.mu.Unlock()
	for key := range r.cache {
		if strings.HasPrefix(key, prefix) {
			delete(r.cache, key)
		}
	}
}

func (r *DingTalkScopeResolver) get(key string) (dingTalkScopeTree, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	entry, ok := r.cache[key]
	if !ok {
		return dingTalkScopeTree{}, false
	}
	if time.Now().After(entry.expiresAt) {
		delete(r.cache, key)
		return dingTalkScopeTree{}, false
	}
	return entry.tree, true
}

func (r *DingTalkScopeResolver) put(key string, tree dingTalkScopeTree) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cache[key] = dingTalkScopeCacheEntry{tree: tree, expiresAt: time.Now().Add(dingTalkScopeCacheTTL)}
}

func ClearDingTalkScopeCache(tenantId int) {
	defaultDingTalkScopeResolver.ClearTenant(tenantId)
}

func dingTalkScopeCacheKey(config entmodel.DingTalkConfig) string {
	return fmt.Sprintf("%d:%d:%s", config.TenantId, config.UpdatedAt, strings.TrimSpace(config.SyncScope))
}

func loadDingTalkScopeTree(ctx context.Context, client DingTalkDirectoryClient, appToken string, scope string) (dingTalkScopeTree, error) {
	roots := parseDingTalkSyncScope(scope)
	if len(roots) == 0 {
		roots = []int64{1}
	}
	tree := dingTalkScopeTree{
		Departments: make(map[int64]DingTalkDepartmentInfo),
		Order:       make([]int64, 0),
	}
	for _, root := range roots {
		if root <= 0 {
			return dingTalkScopeTree{}, ErrDingTalkOAuthProviderFailed
		}
		if _, exists := tree.Departments[root]; exists {
			continue
		}
		tree.Departments[root] = DingTalkDepartmentInfo{
			DeptId: root,
			Name:   fmt.Sprintf("DingTalk Department %d", root),
		}
		tree.Order = append(tree.Order, root)
		if err := loadDingTalkScopeChildren(ctx, client, appToken, root, 0, &tree); err != nil {
			return dingTalkScopeTree{}, err
		}
	}
	return tree, nil
}

func loadDingTalkScopeChildren(ctx context.Context, client DingTalkDirectoryClient, appToken string, parentId int64, depth int, tree *dingTalkScopeTree) error {
	if depth >= dingTalkScopeMaxDepth {
		return ErrDingTalkOAuthProviderFailed
	}
	children, err := client.ListSubDepartments(ctx, appToken, parentId)
	if err != nil {
		return err
	}
	for _, child := range children {
		child.Name = strings.TrimSpace(child.Name)
		if child.DeptId <= 0 || child.Name == "" || child.DeptId == parentId {
			return ErrDingTalkOAuthProviderFailed
		}
		if child.ParentId != 0 && child.ParentId != parentId {
			return ErrDingTalkOAuthProviderFailed
		}
		if _, exists := tree.Departments[child.DeptId]; exists || len(tree.Departments) >= dingTalkScopeMaxNodes {
			return ErrDingTalkOAuthProviderFailed
		}
		child.ParentId = parentId
		tree.Departments[child.DeptId] = child
		tree.Order = append(tree.Order, child.DeptId)
		if err := loadDingTalkScopeChildren(ctx, client, appToken, child.DeptId, depth+1, tree); err != nil {
			return err
		}
	}
	return nil
}

func dingTalkScopeDepartmentIds(tree dingTalkScopeTree, departmentIds []int64) []int64 {
	matched := make([]int64, 0, len(departmentIds))
	seen := make(map[int64]struct{}, len(departmentIds))
	for _, departmentId := range departmentIds {
		if departmentId <= 0 {
			continue
		}
		if _, exists := tree.Departments[departmentId]; !exists {
			continue
		}
		if _, exists := seen[departmentId]; exists {
			continue
		}
		seen[departmentId] = struct{}{}
		matched = append(matched, departmentId)
	}
	sort.Slice(matched, func(i, j int) bool { return matched[i] < matched[j] })
	return matched
}
