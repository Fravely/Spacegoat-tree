CREATE DATABASE InventarioProductosDB;
GO

USE InventarioProductosDB;
GO

CREATE TABLE Productos (
    id INT PRIMARY KEY,
    nombre NVARCHAR(100) NOT NULL,
    categoria NVARCHAR(100) NOT NULL,
    precio DECIMAL(10, 2) NOT NULL,
    stock INT NOT NULL
);
GO

INSERT INTO Productos (id, nombre, categoria, precio, stock) VALUES
(101, 'Laptop Lenovo', 'Tecnologia', 2500.00, 12),
(102, 'Mouse Logitech', 'Tecnologia', 85.50, 40),
(103, 'Teclado Mecanico', 'Tecnologia', 180.00, 25),
(104, 'Monitor Samsung', 'Tecnologia', 720.00, 10),
(105, 'Silla Ergonomica', 'Oficina', 450.00, 8),
(106, 'Escritorio', 'Oficina', 600.00, 5),
(107, 'Cuaderno A4', 'Utiles', 12.50, 100),
(108, 'Lapicero Azul', 'Utiles', 2.00, 300),
(109, 'Mochila', 'Accesorios', 120.00, 20),
(110, 'Audifonos', 'Tecnologia', 150.00, 18);
GO

SELECT * FROM Productos;