CREATE TABLE `tsad_tasks` (
    `name` varchar(125) primary key,
    `state` varchar(255),
    `data_source` text,
    `processed_by` varchar(255),
    `lock_expiration` timestamp NULL DEFAULT '2000-01-01 00:00:00',
    
    UNIQUE INDEX uniq_task (`name`)
) ENGINE=InnoDB CHARSET=utf8; 

CREATE TABLE `tsad_detectors` (
    `host` varchar(125) primary key,
    `num_tasks` int,
    `heart_beat` timestamp NULL DEFAULT '2000-01-01 00:00:00',

    UNIQUE INDEX uniq_detector (`host`)
) ENGINE=InnoDB CHARSET=utf8; 

CREATE TABLE `tsad_duty_lock` (
    `lock_key` varchar(100) primary key,
    `locked_by` varchar(100) ,
    `lock_expiration` timestamp NULL DEFAULT '2000-01-01 00:00:00',
    
    UNIQUE INDEX uniq_key (`lock_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `tsad_model_data` (
    `src_key` varchar(100) primary key,
    `name` varchar(100),
    `data` TEXT,
    `stamp` timestamp NULL DEFAULT '2000-01-01 00:00:00',

    UNIQUE INDEX uniq_key (`src_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;

CREATE TABLE `tsad_points` (
    `src_key` varchar(100) primary key,
    `points` TEXT,

    UNIQUE INDEX uniq_key (`src_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8;