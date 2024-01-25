package web

import (
	"fmt"
	"regexp"
	"strings"
	"web/handler"
	"web/middleware"
)

func (n *node) childOfNotStatic() (*node, bool) {
	if n.regChild != nil {
		return n.regChild, true
	}
	if n.paramChild != nil {
		return n.paramChild, true
	}
	// 如果为 通配符路由 且没有字节点路由，则返回自己
	//if n.typ == nodeTypeAny {
	//	return n, true
	//}
	return n.starChild, n.starChild != nil
}

// child 返回子节点
// 第一个返回值 *node 是命中的节点
// 第二个返回值 bool 代表是否是命中参数路由
// 第三个返回值 bool 代表是否命中
func (n *node) childOf(path string) (*node, bool) {

	if n.children == nil {
		return n.childOfNotStatic()
	}
	root, ok := n.children[path]
	if !ok {
		return n.childOfNotStatic()
	}
	return root, ok

}

func (n *node) notChild() bool {
	if n.children == nil && n.regChild == nil &&
		n.paramChild == nil && n.starChild == nil {
		return true
	}
	return false
}

// findRoute 查找对应的节点
// 注意，返回的 node 内部 HandleFunc 不为 nil 才算是注册了路由
func (r *router) findRoute(method, path string) (*matchInfo, bool) {
	root, ok := r.trees[method]
	if !ok {
		return nil, false
	}
	if path == "/" {
		return &matchInfo{n: root, mdls: root.mdls}, true
	}
	// segs := strings.Split(path[1:], "/")
	segs := strings.Split(strings.Trim(path, "/"), "/")
	mi := &matchInfo{}
	cur := root
	for _, s := range segs {
		// var matchParam bool
		if s == "" {
			return nil, false
		}
		//root, matchParam, ok = root.childOf(s)
		cur, ok = cur.childOf(s)
		if !ok {
			return nil, false
		}
		if cur.typ == nodeTypeReg {
			subMatch := cur.regExpr.MatchString(s)
			if subMatch {
				mi.addValue(cur.paramName, s)
			} else {
				return nil, false
			}
		}
		if cur.typ == nodeTypeParam {
			mi.addValue(cur.paramName, s)
		}
		if cur.typ == nodeTypeAny && cur.notChild() {
			break
		}
	}

	mi.n = cur
	mi.mdls = cur.matchMdls

	//if cur.matchMdls != nil {
	//	mi.mdls = cur.matchMdls
	//} else {
	//	mi.mdls = r.findMdls(root, segs)
	//	cur.matchMdls = mi.mdls
	//}
	return mi, true
}

// 静态路由创建
func (n *node) staticRouteOrCreate(path string) *node {
	if n.children == nil {
		n.children = make(map[string]*node)
	}
	child, ok := n.children[path]
	if !ok {
		child = &node{typ: nodeTypeStatic, path: path}
		n.children[path] = child
	}
	return child
}

// 通配符路由创建
func (n *node) starRouteOrCreate(path string) *node {
	if n.starChild == nil {
		n.starChild = &node{typ: nodeTypeAny, path: path}
	}
	return n.starChild
}

// 参数路由创建
func (n *node) paramRouteOrCreate(path string) *node {
	if n.paramChild == nil {
		n.paramChild = &node{typ: nodeTypeParam, path: path, paramName: path[1:]}
	}
	return n.paramChild
}

// 正则路由创建
func (n *node) regexRouteOrCreate(path string) *node {
	if n.regChild == nil {
		segs := strings.Split(path, "(")
		if len(segs) != 2 {
			panic(fmt.Sprintf("web: 非法路由，不符合正则规范, 必须是 :name(你的正则)的格式 [%s]", path))
		}
		paramName := segs[0][1:]
		reg := regexp.MustCompile("(" + segs[1])
		if reg == nil {
			panic(fmt.Sprintf("web: 非法路由，正则预编译对象不能为 nil"))
		}
		n.regChild = &node{path: path, typ: nodeTypeReg, paramName: paramName, regExpr: reg}
	}
	return n.regChild

}

// 通配符路由冲突检测
func (n *node) starRouteConflict(path string) (string, bool) {
	if n.paramChild != nil {
		return fmt.Sprintf("web: 非法路由，已有路径参数路由。不允许同时注册通配符路由和参数路由 [%s]", path), true
	}
	if n.regChild != nil {
		return fmt.Sprintf("web: 非法路由，已有正则路由。不允许同时注册通配符路由和正则路由 [%s]", path), true
	}
	return "", false
}

// 参数路由冲突检测
func (n *node) paramRouteConflict(path string) (string, bool) {
	if n.starChild != nil {
		return fmt.Sprintf("web: 非法路由，已有通配符路由。不允许同时注册通配符路由和参数路由 [%s]", path), true
	}
	if n.paramChild != nil && n.paramChild.path != path {
		return fmt.Sprintf("web: 路由冲突，参数路由冲突，已有 %s，新注册 %s", n.paramChild.path, path), true
	}
	if n.regChild != nil {
		return fmt.Sprintf("web: 非法路由，已有正则路由。不允许同时注册通配符路由和正则路由 [%s]", path), true
	}
	return "", false
}

func (n *node) regxRouteConflict(path string) (string, bool) {
	if n.paramChild != nil {
		return fmt.Sprintf("web: 非法路由，已经有路径参数路由，不允许同时注册通配符路由和参数路由 [%s]", path), true
	}
	if n.starChild != nil {
		return fmt.Sprintf("web: 非法路由，已有通配符路由。不允许同时注册通配符路由和参数路由 [%s]", path), true
	}
	if n.regChild != nil && n.regChild.path != path {
		return fmt.Sprintf("web: 非法路由, 重复注册正则路由 [%s]", path), true
	}
	return "", false
}

// childOrCreate 查找子节点，如果子节点不存在就创建一个
// 并且将子节点放回去了 children 中
func (n *node) childOrCreate(path string) *node {

	// 如果为正则路由
	if strings.ContainsAny(path, `()`) {
		panicInfo, ok := n.regxRouteConflict(path)
		if ok {
			panic(panicInfo)
		}
		return n.regexRouteOrCreate(path)
	}

	// 通配符路由
	if path == "*" {
		panicInfo, ok := n.starRouteConflict(path)
		if ok {
			panic(panicInfo)
		}
		return n.starRouteOrCreate(path)
	}

	// 以 : 开头，我们认为是参数路由
	if path[0] == ':' {
		panicInfo, ok := n.paramRouteConflict(path)
		if ok {
			panic(panicInfo)
		}
		return n.paramRouteOrCreate(path)
	}

	return n.staticRouteOrCreate(path)
}

func (r *router) checkLegalPath(path string) {
	if path == "" {
		panic("web: 路由是空字符串")
	}
	if path[0] != '/' {
		panic("web: 路由必须以 / 开头")
	}
	if path != "/" && path[len(path)-1] == '/' {
		panic("web: 路由不能以 / 结尾")
	}
}

func (r *router) findMdls(root *node, segs []string) []middleware.Middleware {
	mdls := make([]middleware.Middleware, 0, 10)
	// 用来存放节点的队列
	if len(root.mdls) > 0 {
		mdls = append(mdls, root.mdls...)
	}
	queue := []*node{root}
	for _, seg := range segs {
		queueLen := len(queue)
		for i := 0; i < queueLen; i++ {
			cur := queue[0]
			curChilds, curMdls := cur.childMldsOf(seg)
			queue = append(queue, curChilds...)
			mdls = append(mdls, curMdls...)
			queue = queue[1:len(queue)]
		}
	}
	return mdls
}

func (r *router) findAndLoadMdls(root *node) {
	// 用来存放节点的队列
	queue := []*node{root}

	for len(queue) > 0 {
		queueLen := len(queue)
		for i := 0; i < queueLen; i++ {
			cur := queue[0]
			if cur.route != "" {
				segs := strings.Split(strings.Trim(cur.route, "/"), "/")
				cur.matchMdls = r.findMdls(root, segs)
			} else {
				cur.matchMdls = cur.mdls
			}
			curChilds := cur.onlyChildNodesOf(cur.path)
			queue = append(queue, curChilds...)
			queue = queue[1:len(queue)]
		}
	}

}

func (n *node) onlyChildNodesOf(path string) []*node {
	res := make([]*node, 0, 10)
	if n.children != nil && len(n.children) > 0 {
		if n.path == path {
			for _, staticNode := range n.children {
				res = append(res, staticNode)
			}
		} else {
			staticNode, ok := n.children[path]
			if ok {
				res = append(res, staticNode)
			}
		}
	}

	if n.regChild != nil {
		res = append(res, n.regChild)
	}
	if n.paramChild != nil {
		res = append(res, n.paramChild)
	}
	if n.starChild != nil {
		res = append(res, n.starChild)
	}

	return res
}

func (n *node) childMldsOf(path string) ([]*node, []middleware.Middleware) {
	res := make([]*node, 0, 10)
	mdls := make([]middleware.Middleware, 0, 10)
	if n.children != nil && len(n.children) > 0 {
		staticNode, ok := n.children[path]
		if ok {
			mdls = append(mdls, staticNode.mdls...)
			res = append(res, staticNode)
		}
	}
	if n.regChild != nil {
		mdls = append(mdls, n.regChild.mdls...)
		res = append(res, n.regChild)
	}
	if n.paramChild != nil {
		mdls = append(mdls, n.paramChild.mdls...)
		res = append(res, n.paramChild)
	}
	if n.starChild != nil {
		mdls = append(mdls, n.starChild.mdls...)
		res = append(res, n.starChild)
	}

	return res, mdls
}

// addRoute 注册路由。
// method 是 HTTP 方法
// path 必须以 / 开始并且结尾不能有 /，中间也不允许有连续的 /
func (r *router) addRoute(method, path string, handleFunc handler.HandleFunc, mdls ...middleware.Middleware) {

	r.checkLegalPath(path)

	root, ok := r.trees[method]
	if !ok {
		// 这是一个全新的 HTTP 方法，创建根节点
		root = &node{path: "/"}
		r.trees[method] = root
	}
	if path == "/" {
		if root.handler != nil {
			panic("web: 路由冲突[/]")
		}
		root.mdls = mdls
		root.handler = handleFunc
		return
	}
	// 开始一段段处理
	segs := strings.Split(path[1:], "/")
	for _, s := range segs {
		if s == "" {
			panic(fmt.Sprintf("web: 非法路由。不允许使用 //a/b, /a//b 之类的路由, [%s]", path))
		}
		root = root.childOrCreate(s)
	}

	if root.handler != nil {
		panic(fmt.Sprintf("web: 路由冲突[%s]", path))
	}
	root.handler = handleFunc
	root.mdls = mdls
	root.route = path
}

func newRouter() router {
	return router{
		trees: make(map[string]*node, 12),
	}
}

type router struct {
	// trees 是按照 HTTP 方法来组织的
	// 如 GET => *node
	trees map[string]*node
}

type nodeType int

const (
	// 静态路由
	nodeTypeStatic = 1
	// 正则路由
	nodeTypeReg = 2
	// 路径参数路由
	nodeTypeParam = 3
	// 通配符路由
	nodeTypeAny = 4
)

// node 代表路由树的节点
// 路由树的匹配顺序是：
// 1. 静态完全匹配
// 2. 通配符匹配
// 这是不回溯匹配
type node struct {
	typ  nodeType
	path string
	//  children 子节点 (静态路由节点)
	// 子节点的 path => node
	children map[string]*node
	// handler 命中路由之后执行的逻辑
	handler handler.HandleFunc
	// 注册在该节点上的 middleware
	mdls []middleware.Middleware
	// 该路由要加载的缓存 middleware
	matchMdls []middleware.Middleware
	// route 到达该节点的完整的路由路径
	route string

	// 通配符 * 表达的节点，任意匹配
	starChild *node

	// 参数路由节点
	paramChild *node

	// 正则路由和参数路由都会使用这个字段
	paramName string

	// 正则表达式路由节点
	regChild *node
	// 正常表达式API
	regExpr *regexp.Regexp
}

type matchInfo struct {
	n          *node
	pathParams map[string]string
	mdls       []middleware.Middleware
}

func (m *matchInfo) addValue(key, value string) {
	if m.pathParams == nil {
		// 大多数情况，参数路径只会有一段
		m.pathParams = make(map[string]string)
	}
	m.pathParams[key] = value
}
