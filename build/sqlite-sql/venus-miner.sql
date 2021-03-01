/*
Navicat SQLite Data Transfer

Source Server         : venus-miner
Source Server Version : 30714
Source Host           : :0

Target Server Type    : SQLite
Target Server Version : 30714
File Encoding         : 65001

Date: 2021-03-01 14:49:38
*/

PRAGMA foreign_keys = OFF;

-- ----------------------------
-- Table structure for miners
-- ----------------------------
DROP TABLE IF EXISTS "main"."miners";
CREATE TABLE miners(
	id INT PRIMARY KEY     NOT NULL,
	addr           VARCHAR(30)    NOT NULL,
	listen_api     VARCHAR(50)    NOT NULL,
	token          VARCHAR(255)    NOT NULL
);

-- ----------------------------
-- Records of miners
-- ----------------------------

-- ----------------------------
-- Table structure for venus_api
-- ----------------------------
DROP TABLE IF EXISTS "main"."venus_api";
CREATE TABLE venus_api(
	id INT PRIMARY KEY     NOT NULL,
	api VARCHAR(50)    NOT NULL,
	token VARCHAR(255)    NOT NULL
);

-- ----------------------------
-- Records of venus_api
-- ----------------------------
