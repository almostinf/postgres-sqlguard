MERGE INTO imaginary_inventory AS target
USING imaginary_inventory_feed AS source
ON target.item_id = source.item_id
WHEN MATCHED THEN
    UPDATE SET quantity = source.quantity
WHEN NOT MATCHED BY SOURCE THEN
    DELETE
WHEN NOT MATCHED THEN
    INSERT (item_id, quantity) VALUES (source.item_id, source.quantity);
