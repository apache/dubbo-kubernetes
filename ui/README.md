# dubbo-admin-ui

> dubbo admin ui based on vuetify
> standard front end project

## Build Setup

前端依赖的所有后端 Swagger API 文档目前存放在 [`hack/swagger/swagger.json`](../hack/swagger/swagger.json) 目录，可参考了解后端 API 详情，基于 API 数据格式可对后端接口进行 mock，实现前端闭环开发。

### 前后端集成测试
接下来是关于如何实现前后端集成测试的，首先是对前端代码打包，然后完成后端代码打包并启动 dubbo-cp 进程。

#### 本地前端打包
有两种方式可实现前端代码打包，可根据本机环境任选一种：

##### 1. 基于 Docker 打包
本地安装 docker 后，在项目根目录直接运行以下命令完成打包：

```shell
make build-ui
```

##### 2. 本地 yarn 构建打包
切换到 `ui` 目录后，按照以下方式重新打包前端代码后并重新启动 Admin 进程。

a. 执行以下命令安装相关依赖包
```shell
yarn
```

b. 构建前端组件

```shell
yarn build
```

c. 拷贝构建结果到 admin 发布包

```shell
rm -rf ../app/dubbo-ui/dist
cp -R ./dist ../app/dubbo-ui/
```

#### 打包并启动后端服务
在项目根目录，执行以下命令，完成后端服务本地打包

```shell
make build-dubbbocp
```

切换到以下目录，运行以下进程：

```shell
cd ./app/dubbo-cp
./dubbo-cp
```

此时，即可打开浏览器访问：`http://localhost:38080/admin`

## 其他

For a detailed explanation on how things work, check out the [guide](http://vuejs-templates.github.io/webpack/) and [docs for vue-loader](http://vuejs.github.io/vue-loader).

managed by [front end maven plugin](https://github.com/eirslett/frontend-maven-plugin)

