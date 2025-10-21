# gobatis 

The fantastic library for golang, aims to be the most powerful and stable ORM.


## Overview

* use xml file to define sql
* support transaction
* Logger
* support insert/update/delete xxxxx returning xxx feature
* support window function
* support all SQL features by defining xml
* use sql/database to execute sql
    - postgres
    - mysql
    - SQLite
    - ...

## Getting Started

```shell
++
 |- statements
    |-- user.xml
    |-- order.xml
 |- main.go
```

```xml user.xml
<mapper type="mysql">
    <!--  SQL segements  -->
    <sql id="userColumns">
        ${alias}.name, ${alias}.department
    </sql>
    
   <!-- 查询用户信息 -->
    <select id="findUser" type="mysql">
        select * from employees
        <where>
            <trim prefix="" prefixOverrides="AND | OR">
                <if test="employee_id != null">
                    AND employee_id in (#{employee_id})
                </if>
                <if test="department != null">
                    AND department = #{department}
                </if>
            </trim>
        </where>
        order by id asc limit 5
    </select>
    
    <!-- 添加用户 -->
    <insert id="addUser">
        insert into employees 
        (<include refid="userColumns" alias="alias" value="employees" />)
        values
        <foreach item="item" collection="list" separator=",">
        (#{item.name}, #{item.department})
        </foreach>
    </insert>
    
    <!-- 删除用户 -->
    <delete id="deleteUser">
        delete from employees where id = #{id}
    </delete>
    
    <!-- 修改用户 -->
    <update id="updateUser">
        update employees set name = #{name}, department = #{department} where id = #{id}
    </update>
</mapper>
```


#### 针对于不同的数据库，需要引入不同的数据库驱动:

```go 
import (
    // used for mysql
	_ "github.com/go-sql-driver/mysql"

    // used for postgres
	_ "github.com/jackc/pgx/v5/stdlib"
	
	// used for posgres
	_ "github.com/lib/pq"
)
```


```go main.go
    type Employees struct {
        EmployeeId       int    `expr:"employee_id"`
        Name             string `expr:"name"`
        Department       int    `expr:"department"`
        PerformanceScore string `expr:"performance_score"`
        Salary           string `expr:"salary"`
    }

   //go:embed statements/*.xml
    var embedFs embed.Fs

    db, err := OpenWithEmbedFs(driverName, dsn, embedFs, `statements`)
    if err != nil {
    	panic(err)
    }
    var ctx = context.TODO()

    var data = map[string]interface{}{
        `employee_id`: []int{38, 39, 40},
        `department`:  2,
    }
    var out []Employees
    if err = db.WithContext(ctx).Mapper(`findUser`).Args(data).Find(&out).Error; err != nil {
        panic(err)
    } else {
        fmt.Printf("%#v\n", out)
    }
```

#### 查询数据用 `Find`

- **select * from xxx**
- **insert into xxx returning xxx**
- **update xxx set xxx where xxx returning xxx**
- **delete from xxx where xxx returning xxx**

```go
var out []any
if err := db.WithContext(ctx).
	Mapper(`findEmployeesByIds`).
	Args(&Gobatis{`ids`:[]int{1,3,5}}).Find(&out).Error; err != nil {
	panic(err)
}
fmt.Printf("%#v\n", out)
```

#### 增删改数据但是不返回查询 用 `Execute`

- **insert into xxx values (xxx)**
- **update xxx set xxx where xxx**
- **delete from xxx where xxx**

```go
if err := db.WithContext(ctx).
	Mapper(`updateEmployeesById`).
	Args(&Gobatis{`id`:1, `name`:`test`}).Execute().Error; err != nil {
	panic(err)
}
```

### XML Grammer

**[ 注意 ]**:
由于 `xml` 规范的原因，以下字符需要转义：

`&` 需要转移为 `&amp;`

`<` 需要转移为 `&lt;`

`>` 需要转移为 `&gt;`

`'` 需要转移为 `&apos;`

`"` 需要转移为 `&quot;`

或者你可以使用 `CDATA` 标签包裹：

`<![CDATA[ 内容 ]]>`

**[ 说明 ]**:
标签内容里面替换规则：

`${xxx}` 会被替换为对应的值

`#{xxx}` 会被替换为预处理变量。

* mapper 标签

```xml
<mapper type="postgres|mysql"></mapper> 
```

一个 `mapper` 标签代表一张表，里面包含整个表的增删改查语句，`mapper` 标签包含 `type` 属性
该属性的值为数据库类型，目前支持 `mysql`、 `postgres`、`sqlite` 数据库类型。
设置了 `mapper` 标签的 `type` 属性后，子元素会自动继承该属性，如果子元素单独设置 `type` 那么以子元素设置的为准.

`mapper` 可以包含以下标签: `sql`, `include`, `select`, `insert`, `update`, `delete`.
标签可以多级嵌套, 但是其他之外的标签会被忽略.

* sql 标签

```xml
<sql id="xxx">name,user,address</sql>
```

`sql` 标签用于定义公用部分内容，如数据表的字段信息、公用窗口函数语句等。标签必须定义 `id` 属性，此属性供 `include`标签的 `refid` 属性引用。
`include`引用之前必须先在 `mapper` 子级定义 `sql` 节点。 不能将 `sql` 节点定义为其他标签的内嵌标签.

* include 标签

```xml
<include refid="xxx" alias="xxx" value=""/>
```

`include` 节点用于引用之前定义的 `sql` 节点。 

`include` 节点必须定义 `refid` 属性  此属性引用之前定义的 `sql` 节点。
支持 `alias` 属性与 `value` 属性，`alias` 属性用于指定字段别名，`value` 属性用于指定字段值。

如下示例：

```xml
<sql id="userColumns">${t}.name, ${t}.department</sql>

<include refid="userColumns" alias="t" value="employees"/>
```

此时 `include` 标签会被替换为 

`employees.name, employees.department`

也就是 `${t}` 被替换为 `employees`

* if 标签

```xml
<if test="id != null">
    id = #{id}
</if>
```

`if` 节点用于判断条件，如果`test`条件满足，那么 `if`标签的文本内容会参与组装，否则忽略。

* elif 标签

```xml
<elif test="id != null and id > 10">
    id = #{id}
</elif>
```

注意定义 `elif` 节点前，必须有 `if` 节点。 不能单独定义 `elif` 标签

`elif` 标签与最近的一个 `if` 节点搭配

`elif` 节点用于判断条件，如果`test`条件满足，那么 `elif`标签的文本内容会参与组装，否则忽略。

* else 标签

```xml
<else>
    department = #{department}
</else>
```

`else`标签定义之前必须定义一个 `if` 节点或者 `elif` 节点。

`else` 标签与最近的一个 `if` 或者 `elif` 搭配

`else` 节点存在的作用就是当前 `if` 或者前置的 `elif` 条件都不满足时，执行 `else` 节点的文本内容，

* choose, when, otherwise 标签

```xml
<choose>
    <when test="id != null">
        id = #{id}
    </when>
    <when test="department != null">
        department = #{department}
    </when>
    <otherwise>
        id = 100
    </otherwise>
</choose>
```

[说明]: 当没有 `choose` 节点时候， `when` 节点的运行与 `if` 作用一致.

`choose` 节点下的 多个 `when` & `otherwise` 类似与程序的 `switch` 语句。

`otherwise` 与最近的一个 `when` 搭配。

* where 标签

```xml
<where>
    id = #{id}
    <if test="department != null">and department = #{department}</if>
</where>
```

`where` 标签用于组装 `where` 语句。当子句有 `and` | `or` 开头的时候 `where` 会自动去掉 `and` | `or`, 并且加上 `where` 前缀


* foreach 标签

```xml
<foreach collection="list" item="item" separator="union">
    select * from employees where id = #{item.id}
</foreach>
```

`collection`: 定义的时候传入变量的变量名称

`item`: 表示循环 `collection` 值的时候单个值的变量名

`separator`: 表示 `foreach` 子句的内容之间用什么分隔.


* trim 标签

```xml
<trim prefix="" prefixOverrides="AND | OR">
    other statements
</trim>
```

`trim`标签用来去掉子句中多余的定义在 `prefixOverrides` 属性的内容

多个内容用 `|` 分隔


最后拼接 `prefix` 属性定义的内容。

## expression

* 支持运算符：

`+`, `-`, `*`, `/`, `%` (取模), `^` 或者 `**` (次方)

* 比较运算符：

`==`, `!=`, `<`, `>`, `<=`, `>=`

* 逻辑运算符：

`not` or `!`, `and`, `or` or `||`

* 条件运算符：

`?:` (三元运算符), `??` (nil合并), if else

* 成员运算符：

`[]`, `.`, `?.`, `in`

* 字符串运算符：

`+` (连接字符串), `contains`, `startsWith`, `endsWith`

* 范围运算符：

`..` (范围运算符)

* 切片运算符：

`[:]` (切片运算符)

* 管道运算符：

`|`

#### expr 函数篇

* 字符串函数
  - trim(str[, chars])
  - trimPrefix(str, prefix)
  - trimSuffix(str, suffix)
  - upper(str)
  - lower(str)
  - split(str, delimiter[, n])
  - splitAfter(str, delimiter[, n])
  - replace(str, old, new)
  - repeat(str, n)
  - indexOf(str, substring)
  - lastIndexOf(str, substring)
  - hasPrefix(str, prefix)
  - hasSuffix(str, suffix)

* 日期函数
  - now()
  - duration(str)  有效时间单位 "ns", "us" (or "µs"), "ms", "s", "m", "h".
  - date(str[, format[, timezone]])
  - timezone(str)

* 数字函数
  - max(n1, n2)
  - min(n1, n2)
  - abs(n)
  - ceil(n)
  - floor(n)
  - round(n)

* 数组函数
  - all(array, predicate)
  - any(array, predicate)
  - one(array, predicate)
  - none(array, predicate)
  - map(array, predicate)
  - filter(array, predicate)
  - find(array, predicate)
  - findIndex(array, predicate)
  - findLast(array, predicate)
  - findLastIndex(array, predicate)
  - groupBy(array, predicate)
  - count(array[, predicate])
  - concat(array1, array2[, ...])
  - flatten(array)
  - uniq(array)
  - join(array[, delimiter])
  - reduce(array, predicate[, initialValue])
  - sum(array[, predicate])
  - mean(array)
  - median(array)
  - first(array)
  - last(array)
  - take(array, n)
  - reverse(array)
  - sort(array[, order])
  - sortBy(array[, predicate, order])

* 字典Map函数
  - keys(map)
  - values(map)

* 类型检查转换函数
  - type(v) nil bool int uint float string array map.
  - int(v)
  - float(v)
  - string(v)
  - toJSON(v)
  - fromJSON(v)
  - toBase64(v)
  - fromBase64(v)
  - toPairs(map)
  - fromPairs(array)

* 其他函数
  - len(v) 获取数组、map、字符串的长度
  - get(v, index)

* 位运算函数
  - bitand(int, int)
  - bitor(int, int)
  - bitxor(int, int)
  - bitnand(int, int)
  - bitnot(int)
  - bitshl(int, int)
  - bitshr(int, int)
  - bitushr(int, int)

* 具体的明细内容参考 expr-lang/expr [Language Definition](https://expr-lang.org/docs/language-definition#float)
