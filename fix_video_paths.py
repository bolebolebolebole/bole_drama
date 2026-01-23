#!/usr/bin/env python3
"""
修复视频数据库记录的local_path字段
通过匹配文件修改时间和数据库创建时间
"""

import sqlite3
import os
from datetime import datetime, timedelta
from pathlib import Path

# 配置
DB_PATH = "./data/drama_generator.db"
VIDEO_DIR = "./data/storage/videos"


def parse_db_timestamp(ts_str):
    """解析数据库时间戳格式: 2026-01-23 10:40:23.7108899 +0800 CST"""
    # 移除时区信息
    ts_str = ts_str.split("+")[0].strip()
    # 处理微秒部分（可能超过6位）
    if "." in ts_str:
        date_part, micro_part = ts_str.rsplit(".", 1)
        # 只保留前6位微秒
        micro_part = micro_part[:6].ljust(6, "0")
        ts_str = f"{date_part}.{micro_part}"
    # 解析时间
    return datetime.strptime(ts_str, "%Y-%m-%d %H:%M:%S.%f")


def main():
    # 连接数据库
    conn = sqlite3.connect(DB_PATH)
    cursor = conn.cursor()

    # 获取所有已完成但没有local_path的视频
    cursor.execute("""
        SELECT id, created_at, video_url 
        FROM video_generations 
        WHERE status = 'completed' 
        AND (local_path IS NULL OR local_path = '')
        ORDER BY created_at ASC
    """)

    videos = cursor.fetchall()
    print(f"Found {len(videos)} videos without local_path")

    # 获取所有视频文件及其修改时间
    video_files = []
    for filename in os.listdir(VIDEO_DIR):
        if filename.lower().endswith(".mp4"):
            filepath = os.path.join(VIDEO_DIR, filename)
            mtime = datetime.fromtimestamp(os.path.getmtime(filepath))
            video_files.append({"name": filename, "mtime": mtime, "matched": False})

    # 按修改时间排序
    video_files.sort(key=lambda x: x["mtime"])
    print(f"Found {len(video_files)} video files in storage")

    # 匹配视频记录和文件
    updated = 0
    failed = 0

    for video_id, created_at_str, video_url in videos:
        created_at = parse_db_timestamp(created_at_str)

        # 找到最接近创建时间的文件（在创建时间之后）
        best_match = None
        min_diff = timedelta(days=365)

        for file_info in video_files:
            if file_info["matched"]:
                continue

            # 文件修改时间应该在视频创建时间之后（下载需要时间）
            if file_info["mtime"] > created_at:
                diff = file_info["mtime"] - created_at
                # 通常下载在几秒到几分钟内完成
                if diff < min_diff and diff < timedelta(minutes=10):
                    min_diff = diff
                    best_match = file_info

        if best_match:
            local_path = f"/static/videos/{best_match['name']}"

            # 更新数据库
            try:
                cursor.execute(
                    "UPDATE video_generations SET local_path = ? WHERE id = ?",
                    (local_path, video_id),
                )
                print(
                    f"✓ Video ID {video_id} -> {best_match['name']} (diff: {min_diff})"
                )
                best_match["matched"] = True
                updated += 1
            except Exception as e:
                print(f"✗ Failed to update video {video_id}: {e}")
                failed += 1
        else:
            print(
                f"✗ No matching file found for video ID {video_id} (created at {created_at})"
            )
            failed += 1

    # 提交更改
    conn.commit()
    conn.close()

    # 统计
    print("\n=== Summary ===")
    print(f"Total videos: {len(videos)}")
    print(f"Updated: {updated}")
    print(f"Failed: {failed}")

    unmatched = [f for f in video_files if not f["matched"]]
    print(f"Remaining unmatched files: {len(unmatched)}")

    if unmatched:
        print("\nUnmatched files:")
        for f in unmatched:
            print(f"  - {f['name']} (modified: {f['mtime']})")


if __name__ == "__main__":
    main()
