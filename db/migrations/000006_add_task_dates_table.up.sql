CREATE TABLE task_dates (
    id INT AUTO_INCREMENT PRIMARY KEY,
    task_id INT NOT NULL,
    date DATE NOT NULL,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    UNIQUE(task_id, date)
);
