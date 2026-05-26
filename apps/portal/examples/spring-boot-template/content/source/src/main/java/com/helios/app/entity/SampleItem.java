package com.helios.app.entity;

import jakarta.persistence.Column;
import jakarta.persistence.Entity;
import jakarta.persistence.GeneratedValue;
import jakarta.persistence.GenerationType;
import jakarta.persistence.Id;
import jakarta.persistence.Table;

import java.time.Instant;

/**
 * Sample JPA entity demonstrating database connectivity.
 *
 * <p>This entity exists so the scaffolded project proves end-to-end
 * database integration out of the box. Developers should replace or
 * extend it with their domain models.</p>
 */
@Entity
@Table(name = "sample_items")
public class SampleItem {

    @Id
    @GeneratedValue(strategy = GenerationType.IDENTITY)
    private Long id;

    @Column(nullable = false)
    private String name;

    private String description;

    @Column(name = "created_at", nullable = false, updatable = false)
    private Instant createdAt = Instant.now();

    // ── Constructors ───────────────────────────────────

    protected SampleItem() {
        // JPA requires a no-arg constructor
    }

    public SampleItem(String name, String description) {
        this.name = name;
        this.description = description;
    }

    // ── Getters & Setters ──────────────────────────────

    public Long getId() {
        return id;
    }

    public String getName() {
        return name;
    }

    public void setName(String name) {
        this.name = name;
    }

    public String getDescription() {
        return description;
    }

    public void setDescription(String description) {
        this.description = description;
    }

    public Instant getCreatedAt() {
        return createdAt;
    }
}
