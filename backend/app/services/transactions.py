from sqlalchemy.orm import Session, joinedload

from app.models import Item, Stock, TrackMode, Transaction, TransactionType
from app.schemas import TransactionCreate
from app.services.items import get_item_by_barcode


class TransactionError(Exception):
    def __init__(self, code: str, message: str):
        self.code = code
        self.message = message
        super().__init__(message)


def apply_transaction(db: Session, data: TransactionCreate) -> Transaction:
    item = get_item_by_barcode(db, data.barcode)
    if not item or not item.active:
        raise TransactionError("NOT_FOUND", "Item not found")

    if not item.stock:
        item.stock = Stock(item_id=item.id, quantity=0)
        db.add(item.stock)
        db.flush()

    qty = data.quantity
    if item.track_mode == TrackMode.SNP.value and data.type in (
        TransactionType.OUT.value,
        TransactionType.IN.value,
        TransactionType.RETURN.value,
    ):
        if qty != 1:
            qty = 1

    if data.type != TransactionType.ADJUST.value and qty < 1:
        raise TransactionError(
            "INVALID_QUANTITY",
            "Quantity must be at least 1",
        )

    stock = item.stock
    ttype = data.type

    if ttype == TransactionType.OUT.value:
        if stock.quantity < qty:
            raise TransactionError(
                "INSUFFICIENT_STOCK",
                f"Insufficient stock (have {stock.quantity}, need {qty})",
            )
        stock.quantity -= qty
    elif ttype in (TransactionType.IN.value, TransactionType.RETURN.value):
        stock.quantity += qty
    elif ttype == TransactionType.ADJUST.value:
        stock.quantity = qty
    else:
        raise TransactionError("INVALID_TYPE", "Invalid transaction type")

    tx = Transaction(
        type=ttype,
        item_id=item.id,
        quantity=qty,
        operator_id=data.operator_id,
        note=data.note or "",
    )
    db.add(tx)
    db.commit()
    db.refresh(tx)
    return tx
