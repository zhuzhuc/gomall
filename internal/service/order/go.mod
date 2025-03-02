module github.com/bytedance-youthcamp/demo/internal/service/order

go 1.24

require (
	github.com/bytedance-youthcamp/demo/api/order v0.0.0
	github.com/bytedance-youthcamp/demo/api/cart v0.0.0
	github.com/bytedance-youthcamp/demo/api/product v0.0.0
	github.com/bytedance-youthcamp/demo/api/user v0.0.0
	github.com/bytedance-youthcamp/demo/internal/config v0.0.0
	github.com/bytedance-youthcamp/demo/internal/repository v0.0.0
)

replace (
	github.com/bytedance-youthcamp/demo/api/order => ../../../api/order
	github.com/bytedance-youthcamp/demo/api/cart => ../../../api/cart
	github.com/bytedance-youthcamp/demo/api/product => ../../../api/product
	github.com/bytedance-youthcamp/demo/api/user => ../../../api/user
	github.com/bytedance-youthcamp/demo/internal/config => ../../config
	github.com/bytedance-youthcamp/demo/internal/repository => ../../repository
)
