package reports

import (
	"fmt"
	"time"

	"github.com/jung-kurt/gofpdf"
	"github.com/yourusername/logwatch/internal/storage"
)

func GeneratePDF(stats *storage.Stats, paths []storage.TopEntry, ips []storage.TopEntry, threats []storage.SecurityEvent, filename string) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 16)

	// Title
	pdf.Cell(40, 10, "LogWatch Report")
	pdf.Ln(12)

	// Date
	pdf.SetFont("Arial", "", 12)
	pdf.Cell(40, 10, fmt.Sprintf("Generated: %s", time.Now().Format("2006-01-02 15:04:05")))
	pdf.Ln(10)

	// Stats
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(40, 10, "Statistics (Last 24h)")
	pdf.Ln(8)

	pdf.SetFont("Arial", "", 10)
	pdf.Cell(50, 6, fmt.Sprintf("Total Requests: %d", stats.TotalRequests))
	pdf.Ln(6)
	pdf.Cell(50, 6, fmt.Sprintf("Unique IPs: %d", stats.UniqueIPs))
	pdf.Ln(6)
	pdf.Cell(50, 6, fmt.Sprintf("Total Bytes: %d", stats.TotalBytes))
	pdf.Ln(6)
	pdf.Cell(50, 6, fmt.Sprintf("Error Rate: %.2f%%", stats.ErrorRate))
	pdf.Ln(10)

	// Top Paths
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(40, 10, "Top Paths")
	pdf.Ln(8)

	pdf.SetFont("Arial", "", 10)
	for i, p := range paths {
		if i >= 10 {
			break
		}
		pdf.Cell(100, 6, p.Key)
		pdf.Cell(20, 6, fmt.Sprintf("%d", p.Count))
		pdf.Ln(6)
	}
	pdf.Ln(10)

	// Top IPs
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(40, 10, "Top IPs")
	pdf.Ln(8)

	pdf.SetFont("Arial", "", 10)
	for i, ip := range ips {
		if i >= 10 {
			break
		}
		pdf.Cell(50, 6, ip.Key)
		pdf.Cell(20, 6, fmt.Sprintf("%d", ip.Count))
		pdf.Ln(6)
	}
	pdf.Ln(10)

	// Security Threats
	pdf.SetFont("Arial", "B", 14)
	pdf.Cell(40, 10, "Recent Threats")
	pdf.Ln(8)

	pdf.SetFont("Arial", "", 8)
	for i, t := range threats {
		if i >= 20 {
			break
		}
		pdf.Cell(30, 5, t.IP)
		pdf.Cell(40, 5, t.ThreatType)
		pdf.Cell(20, 5, t.Severity)
		pdf.Cell(10, 5, fmt.Sprintf("%d", t.Score))
		pdf.Ln(5)
	}

	return pdf.OutputFileAndClose(filename)
}