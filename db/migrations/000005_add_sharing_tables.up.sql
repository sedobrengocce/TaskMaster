CREATE TABLE IF NOT EXISTS shared_projects (
    id INT AUTO_INCREMENT PRIMARY KEY,
    project_id INT NOT NULL,
    shared_with_user_id INT NOT NULL,
    FOREIGN KEY (project_id) REFERENCES projects(id) ON DELETE CASCADE,
    FOREIGN KEY (shared_with_user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE (project_id, shared_with_user_id)
); 

CREATE TABLE IF NOT EXISTS shared_tasks (
    id INT AUTO_INCREMENT PRIMARY KEY,
    task_id INT NOT NULL,
    shared_with_user_id INT NOT NULL,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (shared_with_user_id) REFERENCES users(id) ON DELETE CASCADE,
    UNIQUE (task_id, shared_with_user_id)
);
