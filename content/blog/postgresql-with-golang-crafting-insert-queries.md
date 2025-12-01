---
title: Using Text/Templates in Golang To Build SQL Queries
description: Stormkit's approach of eschewing ORM and query builders in favor of raw SQL queries using text/templates.
date: 2024-01-21
---

In this blog post, we will show how we are creating insert queries for PostgreSQL using Golang's text/template package. By sidestepping the abstractions of ORM and query builders, we'll explore how this approach provides greater flexibility, control and a deeper understanding of the underlying database interactions.

ORM solutions, such as [Gorm](https://gorm.io/index.html) abstract away the complexities of SQL queries, enabling developers to interact with databases using familiar programming constructs like structs and methods. Here is a small example

```go
type User struct {
	ID   uint
	Name string
	Age  uint
}

// parts of the code is omitted to make it short
func main() {
	dsn := "user=gorm dbname=gorm"
	db, _ := gorm.Open(postgres.Open(dsn), &gorm.Config{})

  // select * from users
	var users []User
	db.Find(&users)

	// print users in loop
}
```

On the other hand, query builders exemplified by libraries like [sqlx](https://github.com/jmoiron/sqlx) or [squirrel](https://github.com/Masterminds/squirrel), offer a middle ground. They provide a programmatic interface to construct SQL queries dynamically, allowing developers to build queries in a more structured way compared to raw SQL strings.

```go
//you need to annotate your struct
type User struct {
	ID   uint   `db:"id"`
	Name string `db:"name"`
	Age  uint   `db:"age"`
}

func main() {
	db, _ := sql.Open("postgres", "user=squirrel dbname=squirrel sslmode=disable")

	// create a new user
	newUser := User{Name: "Bob Smith", Age: 28}
	insertBuilder := squirrel.
		Insert("users").
		Columns("name", "age").
		Values(newUser.Name, newUser.Age).
		Suffix("RETURNING id")

	var userID uint
	_ = insertBuilder.RunWith(db).QueryRow().Scan(&userID)

	// select all users
	selectBuilder := squirrel.Select("*").From("users")
	rows, _ := selectBuilder.RunWith(db).Query()

	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
    rows.Scan(&user.ID, &user.Name, &user.Age)
		users = append(users, user)
	}

	// print users in loop
}
```

They both have advantages and disadvantages, in Stormkit we use neither of ORM or query builder. We write our own SQL queries and use Go’s text templates to create SQL queries. There are few reasons why we went to this route.

- With numerous libraries available, it becomes a significant effort to evaluate and choose the most stable one with the least number of bugs.
- ORM’s are great most of the time but sometimes there are edge cases that could take a lot of time to figure out and you always need to write raw queries at some stage. Thats why they have api for running raw SQL query
- Every library introduces a learning curve
- Database interaction is spread out throughout a codebase, making it challenging to switch libraries once a choice is made. If we decide to shift away from a library due to dissatisfaction or changing requirements, the transition can be cumbersome and may necessitate significant code modifications.
- Unlike some languages with mature and battle tested ORMs, the Golang ecosystem has yet to settle on a dominant standard. In contrast to languages like [Ruby](https://www.ruby-lang.org/en/), where well established [ActiveRecord](https://www.martinfowler.com/eaaCatalog/activeRecord.html) in [Rails](https://rubyonrails.org/) have stood the test of time, the Go community often leans towards recommending the crafting of custom raw SQL queries. This sentiment is reflected in community discussions where many individuals advocate for building tailored solutions instead of relying on existing libraries.

Here is a simplified example for how we do our insert queries,

```go
//helper function to use in templates
//basically generates string like VALUES ($1,$2....)
func generateValues(numberOfFields, rows int) string {
	var builder strings.Builder
	builder.WriteString("VALUES ")

	for row := 0; row < rows; row++ {
		builder.WriteString("(")
		for field := 0; field < numberOfFields; field++ {
			if field > 0 {
				builder.WriteString(", ")
			}
			index := row*numberOfFields + field + 1
			builder.WriteString(fmt.Sprintf("$%d", index))
		}
		if row < rows-1 {
			builder.WriteString("),")
		} else {
			builder.WriteString(")")
		}
	}

	return builder.String()
}

type Person struct {
	Name  string
	Age   int
	Email string
}

//omitted error checks
func main() {
	connStr := "user=your_username dbname=your_database_name password=your_password host=your_host port=your_port sslmode=disable"
	db, _ := sql.Open("postgres", connStr)
	defer db.Close()

	dataToInsert := []Person{
		{"John Doe", 30, "john.doe@example.com"},
		{"Jane Smith", 25, "jane.smith@example.com"},
	}

	queryTemplate := `
		INSERT INTO persons (name, age, email)
		{{generateValues 3 (len .) }}
		`
	tmpl, _ := template.New("query").Funcs(template.FuncMap{"generateValues": generateValues}).Parse(queryTemplate)

	var queryBuilder strings.Builder

    tmpl.Execute(&queryBuilder, dataToInsert)

    sqlQuery := queryBuilder.String()

	stmt, _ := db.Prepare(sqlQuery)

	defer stmt.Close()

	var values []any

	for _, row := range dataToInsert {
		values = append(values, row.Name, row.Age, row.Email)
	}

	// execute query
    stmt.Exec(values...)
}
```

We had to introduce helper function to use in the template but if you are working with MySQL you won’t even need that since place holders are not numbered.

This approach allows us to create insert queries efficiently, leveraging the simplicity of Golang's text/template package. The use of placeholders ensures protection against SQL injection and by introducing helper functions the code remains concise.

You can easily extend this by adding complex conditions to your insert query or by creating a function that takes an annotated struct and generates the insert query.
