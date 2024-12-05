package main

import (
	"database/sql"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	_ "github.com/mattn/go-sqlite3"
)

// Certificate represents the simplified certificate data structure.
type Certificate struct {
	SerialNumber string `json:"serial_number"`
	Status       string `json:"status"`
}

// Server holds the database connection and other shared resources.
type Server struct {
	DB *sql.DB
}

// initializeDatabase sets up the database and schema.
func initializeDatabase(db *sql.DB) error {
	createTableSQL := `
    CREATE TABLE IF NOT EXISTS certificates (
        serial_number TEXT PRIMARY KEY,
        status TEXT NOT NULL
    );`

	_, err := db.Exec(createTableSQL)
	return err
}

// addCertificate adds a new certificate to the database.
func addCertificate(db *sql.DB, cert Certificate) error {
	insertSQL := `
    INSERT INTO certificates (serial_number, status)
    VALUES (?, ?);`

	_, err := db.Exec(insertSQL, cert.SerialNumber, cert.Status)
	return err
}

// getCertificate retrieves a certificate by serial number.
func getCertificate(db *sql.DB, serialNumber string) (*Certificate, error) {
	querySQL := `
    SELECT serial_number, status FROM certificates WHERE serial_number = ?;`

	row := db.QueryRow(querySQL, serialNumber)

	var cert Certificate
	err := row.Scan(&cert.SerialNumber, &cert.Status)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // Certificate not found
		}
		return nil, err
	}
	return &cert, nil
}

// updateCertificateStatus updates the status of a certificate.
func updateCertificateStatus(db *sql.DB, serialNumber, status string) error {
	updateSQL := `
    UPDATE certificates SET status = ? WHERE serial_number = ?;`

	result, err := db.Exec(updateSQL, status, serialNumber)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("certificate not found")
	}
	return nil
}

// AddCertificateHandler handles the addition of new certificates.
func (s *Server) AddCertificateHandler(c *gin.Context) {
	var cert Certificate
	if err := c.ShouldBindJSON(&cert); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check if certificate already exists
	existingCert, err := getCertificate(s.DB, cert.SerialNumber)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check certificate"})
		return
	}
	if existingCert != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "Certificate already exists"})
		return
	}

	// Add certificate
	err = addCertificate(s.DB, cert)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add certificate"})

		return
	}

	c.JSON(http.StatusCreated, cert)
}

// GetCertificateHandler handles fetching a certificate by serial number.
func (s *Server) GetCertificateHandler(c *gin.Context) {
	serialNumber := c.Param("serial_number")

	// Get certificate
	cert, err := getCertificate(s.DB, serialNumber)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get certificate"})
		return
	}
	if cert == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Certificate not found"})
		return
	}

	c.JSON(http.StatusOK, cert)
}

// UpdateCertificateStatusHandler handles updating the status of a certificate.
func (s *Server) UpdateCertificateStatusHandler(c *gin.Context) {
	serialNumber := c.Param("serial_number")
	var statusUpdate struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&statusUpdate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update status
	err := updateCertificateStatus(s.DB, serialNumber, statusUpdate.Status)
	if err != nil {
		if err.Error() == "certificate not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Certificate not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update certificate status"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Status updated successfully"})
}

func main() {
	// Open database connection
	db, err := sql.Open("sqlite3", "./certificates.db")
	if err != nil {
		log.Fatal("Failed to open database:", err)
	}
	defer db.Close()

	// Initialize database schema
	err = initializeDatabase(db)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	// Initialize the Gin router
	router := gin.Default()

	// Create server instance with the database connection
	server := &Server{DB: db}

	// Define routes
	router.POST("/certificates", server.AddCertificateHandler)
	router.GET("/certificates/:serial_number", server.GetCertificateHandler)
	router.PUT("/certificates/:serial_number/status", server.UpdateCertificateStatusHandler)

	// Start the server on port 8080
	router.Run(":8080")
}
