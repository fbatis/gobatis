# gobatis 

The fantastic library for golang, aims to be the most powerful and stable ORM.


## Overview

* 支持 go1.18+
* 使用 xml 定义 SQL
* 丰富的 Postgres 类型支持
* 支持原生 SQL 执行
* 支持事务
* 支持日志 Logger 接口
* 支持所有SQL特性, 避免常规ORM对于复杂SQL的编写复杂度
* 支持 Postgres 类型的 INSERT/UPDATE/DELETE xxx RETURNING xxx 语法
* 支持 window 窗口函数
* 使用 sql/database 接口
    - Postgres
    - MySQL
    - SQLite

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

    <!--复杂的例子-->
    <select id="findMOrderComplexMore">
        select count(*) as cnt from
        <foreach collection="conditions" item="item" separator="," index="index">
            (select count(*) as cnt from app_events where
            <foreach collection="item.detail" item="detail" separator="and" index="index1">
                <trim prefixOverrides="and">
                  <if test="detail.id != null">
                  and (id = #{detail.id} and id > #{index})
                </if>
                <if test="detail.price != null">
                  and (price = #{detail.price})
                </if>
                <if test="detail.order_id != null or index1 > 0">
                  and (order_id = #{detail.order_id})
                </if>
              </trim>
            </foreach>
            ) as ${item.alias+string(index)}
        </foreach>
        <where>
            (a0.cnt > 0 or b1.cnt > 0)
        </where>
        ${page}
    </select>
    
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
        EmployeeId       int    `json:"employee_id"`
        Name             string `json:"name"`
        Department       int    `json:"department"`
        PerformanceScore string `json:"performance_score"`
        Salary           string `json:"salary"`
    }

   //go:embed statements/*.xml
    var embedFs embed.Fs

    db, err := OpenWithEmbedFs(driverName, dsn, embedFs, `statements`)
    if err != nil {
    	panic(err)
    }
    var ctx = context.TODO()

    var out []Employees
    if err = db.WithContext(ctx).Mapper(`findUser`).Args(&gobatis.Args{
        `employee_id`: []int{38, 39, 40},
        `department`:  2,
    }).Find(&out).Error; err != nil {
        panic(err)
    } else {
        fmt.Printf("%#v\n", out)
    }
```

#### 查询数据用 `Find` 方法

- **select * from xxx** 直接查询
- **insert into xxx returning xxx** 插入后返回插入数据
- **update xxx set xxx where xxx returning xxx** 更新后返回更新数据
- **delete from xxx where xxx returning xxx** 删除后返回删除数据

```go
var out []any
if err := db.WithContext(ctx).
	Mapper(`findEmployeesByIds`).
	Args(&Gobatis{`ids`:[]int{1,3,5}}).Find(&out).Error; err != nil {
	panic(err)
}
fmt.Printf("%#v\n", out)
```

#### 增删改数据但是不返回查询 用 `Execute` 方法

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

#### 原生SQL的增删改查

```go
// 查询操作
var three MOrder
err := tx.RawQuery(`select * from m_order where id = ?`, 3).Find(&three).Error
if err != nil {
	return err
}

// 增删改操作
err := tx.RawExec(`delete from m_order where id = ?`, 3).Execute().Error
if err != nil {
	return err
}	
```

#### 事务用法 
```go
err := db.WithContext(ctx).Transaction(func(tx *gobatis.DB) error {
	// insert
	ndb := tx.Mapper(`insertMOrderByValue`).Args(&gobatis.Args{
		`name`:       `test-value`,
		`price`:      1000,
		`created_at`: time.Now().Add(time.Hour * 24 * -1),
		`updated_at`: time.Now().Add(time.Hour * 6),
	}).Execute()
	if ndb.Error != nil {
		return ndb.Error
	}

	id := ndb.LastInserId

	// find
	var two MOrder
	if err = tx.Mapper(`findMOrderById`).Args(&gobatis.Args{`id`: id}).Find(&two).Error; err != nil {
		return err
	}
	
	return nil
})
if err != nil {
	panic(err)
}
```


## 结构体映射

```sql
create table example (
    id int primary key auto_increment,
    name varchar(255),
    age int
    address varchar(255)
);
```

```go
type Example struct {
	Id     int    `json:"id"`
	Name   string `json:"name"`
	Age    int    `json:"age"`
	Address string `json:"address"`
}
```

正如上面看到的那样, 我们可以使用 `json` **tag** 来让结构体与数据库字段进行映射, 对应的我们也支持更多 `tag`:

按照顺序解析 `json`, `sql`, `db`, `expr`, `leopard`, `gorm` **tag**，遇到了就作为名称映射，否则使用结构体字段名. 建议大家使用 `json` **tag** 即可, 不用增加复杂度。

**[注意]:**

**gobatis** 支持内嵌结构体映射。

```go
type BoxCommon struct {
	Id int `sql:"id"`
}

type BoxData struct {
	BoxCommon
}

type MOrder struct {
	BoxData
	Name      string     `json:"name"`
	Price     float64    `json:"price"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at"`
}
```

**MOrder** 的内嵌结构体 **BoxData** 和 **BoxCommon**结构体。


## Postgres 复杂类型支持

目前 `gobatis` 支持 `Postgres` 的多种复杂类型包括：

| Postgres类型                                                                | gobatis类型              | 解析与回写 |
|---------------------------------------------------------------------------|------------------------|-------|
| int[], integer[], int2[], int4[], int8[]                                  | gobatis.PgArrayInt     | √     |
| text[], varchar[], char[]                                                 | gobatis.PgArrayString  | √     |
| bool[]                                                                    | gobatis.PgArrayBool    | √     |
| float[], float4[], float8[]                                               | gobatis.PgArrayFloat   | √     |
| record[]                                                                  | gobatis.PgArrayRecord  | √     |
| line[]                                                                    | gobatis.PgArrayLine    | √     |
| point[]                                                                   | gobatis.PgArrayPoint   | √     |
| polygon[]                                                                 | gobatis.PgArrayPolygon | √     |
| circle[]                                                                  | gobatis.PgArrayCircle  | √     |
| box[]                                                                     | gobatis.PgArrayBox     | √     |
| path[]                                                                    | gobatis.PgArrayPath    | √     |
| lseg[]                                                                    | gobatis.PgArrayLSeg    | √     |
| int4range[], int8range[], numrange[], tsrange[], tstzrange[], daterange[] | gobatis.PgArrayRange   | √     |
| create type as (xxx) 自定义类型                                                | gobatis.PgRecord       | √     |
| create type as (xxx) 自定义类型数组                                              | gobatis.PgArrayRecord  | √     |
| int4range, int8range, numrange, tsrange, tstzrange, daterange             | gobatis.PgRange        | √     |
| point                                                                     | gobatis.PgPoint        | √     |
| line                                                                      | gobatis.PgLine         | √     |
| polygon                                                                   | gobatis.PgPolygon      | √     |
| lseg                                                                      | gobatis.PgLSeg         | √     |
| box                                                                       | gobatis.PgBox          | √     |
| path                                                                      | gobatis.PgPath         | √     |
| circle                                                                    | gobatis.PgCircle       | √     |
| PostGIS                                                                   | -                      | ×     |


**[ PostGIS ]** 对于 `PostGIS` 中的以下类型的支持后续视情况迭代跟进
- `POINT` 二维 或者 三维 点类型
- `LINESTRING` 线类型
- `POLYGON` 多边形类型
- `MULTIPOINT` 多点类型
- `MULTILINESTRING` 多线类型
- `MULTIPOLYGON`  多多边形类型
- `GEOMETRYCOLLECTION`  几何集合类型
- `CIRCULARSTRING`  圆弧字符串类型
- `COMPOUNDCURVE`  复合曲线类型
- `CURVEPOLYGON`  曲线多边形类型

示例如下：

```text
create table example (
    id serial,
    name text,
    cards int[]
);

// Example
type Example struct {
  Id int `json:"id"`
  Name string `json:"name"`
  Cards gobatis.PgArrayInt `json:"cards"`
}

// Query
var out []*Example
err := db.WithContext(ctx).Mapper(`xxx`).Args(&gobatis.Args{`id`: 1}).Find(&out).Error

// Insert & Update & Delete
err := db.WithContext(ctx).Mapper(`xxx`).Args(&gobatis.Args{`id`: 1, `name`: `hello`, `cards`: &gobatis.PgArrayInt{1, 2, 3}}).Execute().Error
```

如果是自定义类型，则需要使用 `gobatis.PgRecord` 或者 `gobatis.PgArrayRecord` 作为结构体的值变量，实现 `sql.Scanner` & `driver.Valuer` 接口即可：

- **gobatis.PgArrayRecord**: 自定义的类型的数组
- **gobatis.PgRecord**: 自定义类型

```text

create type card as (
    card_num text,
    card_name text,
    card_isn varchar(32)
);

create table exmaple (
    id serial,
    name text,
    cards card[]
);

type Card struct {
    CardNum string `json:"card_num"`
    CardName string `json:"card_name"`
    CardIsn string `json:"card_isn"`
}

type Cards struct {
    value gobatis.PgArrayRecord
    Details []*Card
}

func (c *Cards) Scan(value interface{}) error {
    err := c.value.Scan(value)
    if err != nil {
        return err
    }
    for _, cardItem := range c.value {
        if len(cardItem) != 3 {
            return errors.New("card detail is invalid")
        }
        c.Details = append(c.Details, &Card{
            CardNum: cardItem[0],
            CardName: cardItem[1],
            CardIsn: cardItem[2],
        })
    }
    return nil
}

func (c *Cards) Value() (driver.Value, error) {
	c.value = nil
	for _, card := range c.Details {
		c.value = append(c.value, []string{card.CardNum, card.CardName, card.CardIsn})
	}
	return c.value.Value()
}

type Example struct {
  Id int `json:"id"`
  Name string `json:"name"`
  Cards Cards  `json:"cards"`
}

// Query
var out []Example
if err = db.WithContext(ctx).Mapper(`findCardById`).Args(&gobatis.Args{`id`: 2}).Find(&out).Error; err != nil {
    panic(err)
}
t.Log(out)

// Insert & Update & delete
err = db.WithContext(ctx).Mapper(`insertCard`).
		Args(&gobatis.Args{
			`name`: `card example 2`,
			`cards`: &Cards{
				Details: []*Card{
					{
						CardNum:  `CardNum1`,
						CardName: `CardName1`,
						CardIsn:  `CardIsn1`,
					},
					{
						CardNum:  `CardNum2`,
						CardName: `CardName2`,
						CardIsn:  `CardIsn2`,
					},
				},
			},
		}).Execute().Error

```

### 动态SQL

**gobatis** 采用 `xml` 模板实现动态编程得到 `SQL`

**[ 注意 ]**:
由于 `xml` 规范的原因，以下字符需要转义：

`&` 需要转义为 `&amp;`

`<` 需要转义为 `&lt;`

`>` 需要转义为 `&gt;`

`'` 需要转义为 `&apos;`

`"` 需要转义为 `&quot;`

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
`include`引用之前必须先在 `mapper` 子级定义 `sql` 节点。 

**不能将 `sql` 节点定义为其他标签的内嵌标签(特例：执行单元 `insert`, `select`, `delete`, `update` 标签内部只需要在 `include` 标签前定义即可, 既是是内嵌标签也没有问题).**

正确的示例：

```xml
<mapper>
    <sql id="modelColumnsNest">*</sql>  <!-- 正确的位置1 -->
    <insert id="insertMOrderMultiAndReturning">
        <sql id="modelColumnsNest">*</sql>  <!-- 正确的位置2 -->
        insert into m_order (order_id, price, strategy, created_at, updated_at) values
        <foreach item="item" collection="list" separator=",">
          <sql id="modelColumnsNest">*</sql>  <!-- 正确的位置3 -->
            (#{item.OrderId}, #{item.Price}, #{item.Strategy}, #{item.CreatedAt}, #{item.UpdatedAt})
        </foreach>
        returning <include refid="modelColumnsNest"/>  <!-- 此处引用 -->
    </insert>
    <sql id="modelColumnsNest">*</sql>  <!-- 正确的位置4 -->
</mapper>
```

错误的示例：

```xml
<mapper>
    <insert id="insertMOrderMultiAndReturning">
        <sql id="modelColumnsNest">*</sql>  <!-- 错误的位置1 -->
        insert into m_order (order_id, price, strategy, created_at, updated_at) values
        <foreach item="item" collection="list" separator=",">
          <sql id="modelColumnsNest">*</sql>  <!-- 错误的位置2 -->
            (#{item.OrderId}, #{item.Price}, #{item.Strategy}, #{item.CreatedAt}, #{item.UpdatedAt})
        </foreach>
        <sql id="modelColumnsNest">*</sql>  <!-- 错误的位置3 -->
    </insert>

    <select id="findMOrderById">
        <!-- 此处引用 -->
        select <include refid="modelColumnsNest"/> from m_order where id in (#{map(ids, .Id)})
    </select>
</mapper>
```

**[ 注意 ]**:

`mapper` 标签下级的 `sql` 节点, 所有其他地方都可以 `include` 引用, 但是各执行单元(`update`, `insert`, `select`, `delete`) 内的 `sql` 节点, 只能各单元内使用, 只需要遵守 `先定义后使用` 的原则即可.

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
<foreach collection="list" item="item" separator="union" index="index">
    select * from employees where id = #{item.id}
</foreach>
```

`collection`: 定义的时候传入变量的变量名称

`item`: 表示循环 `collection` 值的时候单个值的变量名

`separator`: 表示 `foreach` 子句的内容之间用什么分隔.

`index`: 表示数组的元素的索引，从 `0` 开始, 可选属性


* trim 标签

```xml
<trim prefix="" prefixOverrides="AND | OR">
    other statements
</trim>
```

`trim`标签用来去掉子句中多余的定义在 `prefixOverrides` 属性的内容

多个内容用 `|` 分隔


最后在内容前拼接 `prefix` 属性定义的内容。

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









