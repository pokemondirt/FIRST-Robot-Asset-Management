from datetime import datetime, timedelta

from fastapi import APIRouter, Depends
from sqlalchemy import func
from sqlalchemy.orm import Session, joinedload

from app.database import get_db
from app.models import Item, Stock, Transaction, TransactionType
from app.schemas import (
    DashboardResponse,
    InboundSummary,
    LowStockAlert,
    RecentInboundLine,
)

router = APIRouter(prefix="/api/dashboard", tags=["dashboard"])


def _start_of_today() -> datetime:
    now = datetime.now()
    return now.replace(hour=0, minute=0, second=0, microsecond=0)


def _start_of_week() -> datetime:
    today = _start_of_today()
    return today - timedelta(days=today.weekday())


@router.get("", response_model=DashboardResponse)
def get_dashboard(db: Session = Depends(get_db)):
    start_today = _start_of_today()
    start_week = _start_of_week()

    today_tx = (
        db.query(Transaction)
        .filter(
            Transaction.type == TransactionType.IN.value,
            Transaction.created_at >= start_today,
        )
        .all()
    )
    today_qty = sum(t.quantity for t in today_tx)

    week_qty = (
        db.query(func.coalesce(func.sum(Transaction.quantity), 0))
        .filter(
            Transaction.type == TransactionType.IN.value,
            Transaction.created_at >= start_week,
        )
        .scalar()
    )

    recent = (
        db.query(Transaction)
        .options(
            joinedload(Transaction.item),
            joinedload(Transaction.operator),
        )
        .filter(Transaction.type == TransactionType.IN.value)
        .order_by(Transaction.created_at.desc())
        .limit(15)
        .all()
    )
    recent_lines = [
        RecentInboundLine(
            id=tx.id,
            barcode=tx.item.barcode,
            name_zh=tx.item.name_zh,
            name_en=tx.item.name_en,
            quantity=tx.quantity,
            operator_name=tx.operator.display_name if tx.operator else None,
            created_at=tx.created_at,
        )
        for tx in recent
    ]

    items = (
        db.query(Item)
        .options(joinedload(Item.stock))
        .filter(Item.active.is_(True))
        .all()
    )
    alerts: list[LowStockAlert] = []
    for item in items:
        qty = item.stock.quantity if item.stock else 0
        is_empty = qty <= 0
        is_low = item.min_stock > 0 and qty < item.min_stock
        if not is_empty and not is_low:
            continue
        level = "critical" if is_empty else "warning"
        shortage = max(0, item.min_stock - qty) if item.min_stock > 0 else (1 if is_empty else 0)
        alerts.append(
            LowStockAlert(
                id=item.id,
                barcode=item.barcode,
                name_zh=item.name_zh,
                name_en=item.name_en,
                program=item.program,
                category=item.category,
                quantity=qty,
                min_stock=item.min_stock,
                shortage=shortage,
                unit=item.unit,
                level=level,
            )
        )

    alerts.sort(key=lambda a: (0 if a.level == "critical" else 1, -a.shortage, a.quantity))

    return DashboardResponse(
        generated_at=datetime.now(),
        inbound=InboundSummary(
            today_transactions=len(today_tx),
            today_quantity=int(today_qty),
            week_quantity=int(week_qty or 0),
        ),
        recent_inbound=recent_lines,
        low_stock_alerts=alerts,
        alert_count=len(alerts),
    )
