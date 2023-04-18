# venus-miner changelog

# 1.11.0-rc1 / 2023-04-18

### New Features
* feat: mysql table migrate for miner_blocks / 更新数据库字段为可空或者添加默认值[[#168](https://github.com/filecoin-project/venus-miner/pull/168)]
* feat: update AuthClient which with token /客户端token验证 [[#169](https://github.com/filecoin-project/venus-miner/pull/169)]
* feat: add status api / 添加状态检测接口 [[#172](https://github.com/filecoin-project/venus-miner/pull/172)]
* feat: add docker push /增加推送到镜像仓库的功能 [[#183](https://github.com/filecoin-project/venus-miner/pull/183)]

### Improvements
* opt: chain-forked check /链分叉判断时只对同周期内判断 [[#181](https://github.com/filecoin-project/venus-miner/pull/181)]
* opt: set the select messages timeout / 选择消息设置5秒超时 [[#184](https://github.com/filecoin-project/venus-miner/pull/184)]

### Bug Fixes
* fix:check gateway fail /修复gateway配置检查失败的问题 [[#177](https://github.com/filecoin-project/venus-miner/pull/177)]
* fix: config check / 修复配置检测失败的问题 [[#178]( https://github.com/filecoin-project/venus-miner/pull/178)]
# 1.10.0 / 2023-03-02

## Improvements

- 数据表 `miner_blocks` 字段设置默认值，`winning_at` 允许为 `null`

# 1.10.0-rc1 / 2023-02-17

## New features
- feat: user data isolation / 增加用户数据隔离  (#163) ([filecoin-project/venus-miner#163](https://github.com/filecoin-project/venus-miner/pull/163))


# 1.9.0 / 2022-12-30

## Dependency Updates

- 升级venus组件的版本


# 1.8.0 / 2022-11-16

## Dependency Updates

- github.com/filecoin-project/venus (-> v1.8.0)
- github.com/filecoin-project/venus-auth (-> v1.8.0)


# 1.8.0-rc5 / 2022-11-03

## Improvements

- 增加 `miners`  是否出块的控制开关，需要 `venus-auth` 版本 >= v1.8.0-rc4.


# 1.8.0-rc4 / 2022-10-26

## Fixes

- 修复计算历史出块权时panic

# 1.8.0-rc3 / 2022-10-20

## Improvements

- 不记录没有获胜的出块Timeout

## 注意事项

从 `1.7.*` 升级会自动迁移配置文件，从 `1.6.*` 升级需重新初始化`Repo`(init)


# 1.8.0-rc2 / 2022-10-19

## Improvements

- 配置项 `MySQL.ConnMaxLifeTime` 改为字符窜格式: `60 -> "1m0s"`；
- `PropagationDelaySecs` 和 `MinerOnceTimeout` 由配置文件设置；
- Repo目录增加 `version`用于自动升级配置文件等。

## 注意事项

从 `1.7.*` 升级会自动迁移配置文件，从 `1.6.*` 升级需重新初始化`Repo`(init)



# 1.6.1 / 2022-07-22

## Improvements

- 网络参数从同步节点请求，移除本地配置；
- 简化配置文件，参考 配置文件解析;
- 移除 `venus-shared` 中已有的数据类型;
- 移除没有实际作用的代码;
- 矿工由 `venus-auth` 管理，移除本地的矿工管理模块；
- 移除对 `filecoin-ffi` 的依赖；
- 新增配置项说明文档；
- 新增快速启动文档；
- 优化出块，在创建区块前再次尝试获取上一轮`base`，做如下处理：
  - 获得较优 `base`（满足获胜条件并有更多的区块），则选择后者进行出块，避免获取 `base` 不足引起的孤块； 
  - `base` 不匹配（发生了链分叉，之前获得的 `base` 计算的出块权是无效的），不进行无意义的区块创建。
- 修复采用 `mysql` 数据库时 `parenet_key` 字段长度不足问题。

## 注意事项

升级到此版本涉及配置文件的改动，有两个选择：
- 重新初始化 `repo`；
- 手动修改配置文件（参考config-desc.md）
