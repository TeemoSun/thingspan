"""category feature flags

Revision ID: 2603a2cb4f14
Revises: 666f352524fb
Create Date: 2026-08-09 20:01:27.478340

"""
from typing import Sequence, Union

from alembic import op
import sqlalchemy as sa
from sqlalchemy.dialects import sqlite

revision: str = '2603a2cb4f14'
down_revision: Union[str, None] = '666f352524fb'
branch_labels: Union[str, Sequence[str], None] = None
depends_on: Union[str, Sequence[str], None] = None


def upgrade() -> None:
    # 类别勾选标志：先带默认值加列（SQLite NOT NULL 需要 server_default）
    for col in ("has_warranty", "has_expiry", "can_sell", "can_break", "has_serial", "has_model"):
        op.add_column("categories", sa.Column(col, sa.Boolean(), nullable=False, server_default=sa.false()))
    # 旧模板数据映射：product → 保修/序列号/型号/可售出/可损坏；membership → 到期日期
    op.execute("UPDATE categories SET has_warranty=1, has_serial=1, has_model=1, can_sell=1, can_break=1 WHERE template='product'")
    op.execute("UPDATE categories SET has_expiry=1 WHERE template='membership'")
    op.execute("UPDATE categories SET can_sell=1, can_break=1 WHERE template='other'")
    # 移除废弃列与自定义字段值
    op.drop_column('assets', 'custom_values')
    op.drop_column('categories', 'template')
    op.drop_column('categories', 'fields')


def downgrade() -> None:
    op.add_column('categories', sa.Column('fields', sqlite.JSON(), nullable=False, server_default=sa.text("'[]'")))
    op.add_column('categories', sa.Column('template', sa.VARCHAR(length=20), nullable=False, server_default=sa.text("'other'")))
    op.drop_column('categories', 'has_model')
    op.drop_column('categories', 'has_serial')
    op.drop_column('categories', 'can_break')
    op.drop_column('categories', 'can_sell')
    op.drop_column('categories', 'has_expiry')
    op.drop_column('categories', 'has_warranty')
    op.add_column('assets', sa.Column('custom_values', sqlite.JSON(), nullable=False, server_default=sa.text("'{}'")))
