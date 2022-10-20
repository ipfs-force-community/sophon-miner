# venus-miner changelog

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
