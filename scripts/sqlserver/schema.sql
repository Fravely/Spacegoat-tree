IF DB_ID(N'ScapegoatDemo') IS NULL
BEGIN
	CREATE DATABASE ScapegoatDemo;
END
GO

USE ScapegoatDemo;
GO

IF OBJECT_ID(N'dbo.Productos', N'U') IS NULL
BEGIN
	CREATE TABLE dbo.Productos (
		id INT NOT NULL PRIMARY KEY,
		nombre NVARCHAR(120) NOT NULL,
		categoria NVARCHAR(80) NOT NULL,
		precio DECIMAL(10, 2) NOT NULL,
		stock INT NOT NULL
	);
END
GO
