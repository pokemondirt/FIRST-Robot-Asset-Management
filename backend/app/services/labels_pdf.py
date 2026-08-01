import io
from pathlib import Path

from reportlab.graphics.barcode import code128
from reportlab.lib.units import mm
from reportlab.pdfgen import canvas

from app.config import settings
from app.models import Item


def _get_label_size():
    from app.database import SessionLocal
    from app.models import AppSetting

    w, h = settings.label_width_mm, settings.label_height_mm
    db = SessionLocal()
    try:
        row_w = db.get(AppSetting, "label_width_mm")
        row_h = db.get(AppSetting, "label_height_mm")
        if row_w:
            w = float(row_w.value)
        if row_h:
            h = float(row_h.value)
    finally:
        db.close()
    return w, h


def build_labels_pdf(items: list[Item]) -> bytes:
    width_mm, height_mm = _get_label_size()
    page_w = width_mm * mm
    page_h = height_mm * mm

    buffer = io.BytesIO()
    c = canvas.Canvas(buffer, pagesize=(page_w, page_h))

    for item in items:
        _draw_label(c, item, page_w, page_h)
        c.showPage()

    c.save()
    buffer.seek(0)
    return buffer.read()


def _draw_label(c: canvas.Canvas, item: Item, page_w: float, page_h: float):
    barcode_value = item.barcode
    bar_height = 12 * mm
    margin = 1.5 * mm

    bc = code128.Code128(
        barcode_value,
        barHeight=bar_height,
        barWidth=0.35 * mm,
        humanReadable=False,
    )
    bw = bc.width
    x = max(margin, (page_w - bw) / 2)
    y = page_h - margin - bar_height
    bc.drawOn(c, x, y)

    c.setFont("Helvetica", 6)
    c.drawCentredString(page_w / 2, y - 2.5 * mm, barcode_value)

    label = item.name_en if len(item.name_en) <= 28 else item.name_en[:25] + "..."
    c.setFont("Helvetica", 5)
    c.drawCentredString(page_w / 2, margin, label)


def save_labels_pdf(items: list[Item], path: Path) -> Path:
    data = build_labels_pdf(items)
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_bytes(data)
    return path
